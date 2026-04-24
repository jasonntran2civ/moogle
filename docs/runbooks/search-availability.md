# Runbook: search availability < 99.5%

Triggered by `SearchAvailabilityBelow99_5`.

## Triage

```bash
# Recent 5xx by route
gcloud logging read 'severity>=ERROR AND resource.labels.service_name="gateway"' --limit 50

# scorer-pool health
grpcurl -plaintext scorer:50052 grpc.health.v1.Health/Check

# Are all three indexes reachable from the scorer?
curl -s http://meilisearch:7700/health
curl -s http://qdrant:6333/healthz
echo 'RETURN 1' | cypher-shell -u neo4j -p $NEO4J_PASSWORD -a bolt://neo4j:7687
```

## Common causes & first response

- Gateway crashed: `docker compose restart gateway`. Confirm the WS endpoint comes back up.
- Scorer pool gRPC unreachable: check OTel for which sub-scorer was failing first; restart the affected NAS service.
- One of the three index backends down: traffic still serves with `degraded_components` in `ScorerHealthzResponse`. Bring the backend back; no failover.
- Cloudflare Tunnel disconnect: check Cloudflare dashboard; restart `cloudflared` on TrueNAS.
