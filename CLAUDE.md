# CLAUDE.md

Guidance for Claude Code agents working in this repository.

## Project

EvidenceLens is a free, public, agentic biomedical evidence search engine — a polyglot monorepo of ingesters, processor, embedder, indexer, scorer, gateway, agent, MCP server, and frontend. The complete engineering specification lives at [`../moogle/docs/EVIDENCELENS_SPEC.md`](../moogle/docs/EVIDENCELENS_SPEC.md) (1745 lines). The orchestrator plan that drives all work is at [docs/orchestrator-plan.md](docs/orchestrator-plan.md). Both files override anything in this CLAUDE.md when they conflict.

## Architecture (terse)

```
ingesters (Go, GCP Cloud Run)         → R2 (raw archive)
                                      → GCP Pub/Sub (raw-docs)
                                          ↓ (via pubsub-bridge Worker)
                                      NATS JetStream
                                          ↓
processor (Python, NAS)               → embedder (Python + vLLM, NAS GPU, gRPC)
                                      → open-payments lookup (HTTP)
                                      → NATS (indexable-docs.{source})
                                          ↓
indexer (Go, NAS)                     → Meilisearch (text/facets)
                                      → Qdrant (1024-d vectors)
                                      → Neo4j Community (citation graph)
                                          ↓
scorer-pool (Python, NAS, gRPC)       BM25 / vector / citation / recency → RRF → XGBoost LTR
                                          ↓
gateway (NestJS, Dokploy)             REST + GraphQL + WS + BYOK proxy
agent (Python FastAPI, Dokploy)       BYOK SSE proxy (Anthropic / OpenAI-compat / Ollama)
mcp-server (TS, Dokploy)              MCP v2025-06 over stdio + HTTP+SSE
                                          ↓
frontend (Next.js 15, Cloudflare Pages) — WebSocket-streamed result waves, COI badges, WebLLM
workers (Cloudflare): pubsub-bridge, click-logger, webllm-shard, turnstile-verify
analytics: BigQuery + Firestore (A/B) + interleaving evaluator
```

Data flow is one-directional. NATS is the in-cluster queue + ephemeral state; Postgres is durable state (`ingestion_state`, `recall_events`, `share_links`, `byok_proxy_telemetry`, `ab_assignment_audit`, `author_payment_cache`).

## Conventions

**Env vars** — every service reads connection config from env. Standard names:
- Postgres: `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`
- NATS: `NATS_URL`, `NATS_STREAM`
- Pub/Sub: `GCP_PROJECT`, `PUBSUB_TOPIC`, `PUBSUB_SUBSCRIPTION`
- Redis: `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`
- Meilisearch: `MEILI_URL`, `MEILI_KEY`
- Qdrant: `QDRANT_URL`, `QDRANT_API_KEY`
- Neo4j: `NEO4J_URL`, `NEO4J_USER`, `NEO4J_PASSWORD`
- R2: `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`
- OTel: `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME`

Every service has a `.env.example` listing exactly what it needs. Never commit `.env`.

**Service skeletons** — copy the pattern from [Moogle's spider main.go](../moogle/services/spider/cmd/spider/main.go) for Go and [Moogle's indexer main.py](../moogle/services/indexer/main.py) for Python. Both demonstrate flag parsing, env helpers, signal handling, and graceful shutdown. Apply universally; do not invent variants.

**Inter-service contracts** — typed via Protobuf in [proto/evidencelens/v1/](proto/evidencelens/v1/). Generated stubs in [proto/gen/](proto/gen/) are committed to avoid CI codegen complexity. Any change to `.proto`, `gateway/src/schema.graphql`, or `docs/api/openapi.yaml` requires a PR labeled `rfc-interface`. See [docs/rfcs/README.md](docs/rfcs/README.md).

**Observability** — every service initializes OpenTelemetry on startup. Helpers in `ingest/pkg/otel/` (Go) and `process/utils/otel.py` (Python). Traces + metrics → OTel Collector on TrueNAS → Grafana Cloud (Tempo, Loki, Mimir). Frontend uses Sentry breadcrumbs.

**Testing** — minimum 80% unit coverage gated in CI. Integration tests run against docker-compose. Network-dependent tests use recorded fixtures: [go-vcr](https://github.com/dnaeon/go-vcr) for Go ingesters, [vcrpy](https://github.com/kevin1024/vcrpy) for Python.

## Common commands

```bash
# Repo-wide
make smoke                              # end-to-end docker-compose smoke
pnpm install && pnpm -r build           # all TS workspaces
go work sync && go build ./...          # all Go workspaces
uv sync && uv run pytest                # all Python workspaces (in each python service dir)
buf lint && buf generate                # proto check + codegen

# Per-service: see services' README.md
```

## What NOT to do

- **No MVP phasing.** The orchestrator plan ships the full system at once. Don't degrade scope.
- **No paid services.** Every dependency must be free-tier or self-hosted.
- **No server-side LLM token burn.** Three-tier inference (BYOK / MCP / WebLLM) is non-negotiable.
- **No accessibility regressions.** WCAG 2.2 AA, axe-core 0 violations gated in CI.
- **No skipping OTel.** Every service must export traces + metrics.
- **No proto / contract changes** without an `rfc-interface` PR.
- **No copying Moogle anti-patterns** — see [docs/orchestrator-plan.md](docs/orchestrator-plan.md) for the explicit don't-copy list (stub `monitoring/`, placeholder tests, REST-only RPC, no-OTel, sync-fetch frontend, Python file duplication).

## Reference repository

[`../moogle/`](../moogle/) is the architectural template for service skeletons, env-var config, batching, and concurrency. Treat it as reference only — the EvidenceLens spec wins where they differ. Notably: EvidenceLens uses gRPC + Protobuf (Moogle does not), OTel (Moogle does not), WebSocket-streamed result waves (Moogle uses sync fetch), NATS JetStream (Moogle uses Redis queues).
