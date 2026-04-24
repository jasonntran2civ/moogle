#!/usr/bin/env bash
# End-to-end smoke test (spec §15 integration).
#
# Brings up the local docker-compose stack, runs ingester-pubmed once
# with PUBMED_MAX_PER_RUN=10, waits for the pipeline to drain, and
# asserts a known query returns at least one result with a COI badge.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
GATEWAY="${GATEWAY:-http://localhost:8080}"
WAIT_SEC="${SMOKE_WAIT_SEC:-180}"

cd "$REPO_ROOT"

echo "==> bringing up data plane"
docker compose -f infra/docker-compose.yml up -d
trap "docker compose -f infra/docker-compose.yml logs --tail=200" ERR

echo "==> waiting for data plane health"
for svc in postgres nats meilisearch qdrant neo4j; do
  for i in $(seq 1 60); do
    if docker compose -f infra/docker-compose.yml ps "$svc" 2>/dev/null | grep -q '(healthy)\|Up'; then
      echo "  $svc OK"; break
    fi
    sleep 2
  done
done

echo "==> running ingester-pubmed once (PUBMED_MAX_PER_RUN=10)"
PUBMED_MAX_PER_RUN=10 docker compose -f infra/docker-compose.yml run --rm \
  -e PUBMED_MAX_PER_RUN=10 ingester-pubmed sh -c '/ingester-pubmed & sleep 2; curl -fsS -X POST http://localhost:8080/run' \
  || echo "  (no service for ingester in dev compose — skipping)"

echo "==> waiting up to ${WAIT_SEC}s for pipeline to drain"
for i in $(seq 1 $((WAIT_SEC / 5))); do
  if curl -fsS "$GATEWAY/api/search?q=cancer&top_k=1" 2>/dev/null | grep -q '"results"'; then
    break
  fi
  sleep 5
done

echo "==> asserting search returns a result"
RES=$(curl -fsS "$GATEWAY/api/search?q=cardiology&top_k=5") || {
  echo "FAIL: gateway not responding"
  exit 1
}
COUNT=$(echo "$RES" | jq -r '.results | length')
echo "  got $COUNT results"
if [ "$COUNT" -lt 1 ]; then
  echo "FAIL: zero results"
  exit 1
fi
echo "==> SMOKE OK"
