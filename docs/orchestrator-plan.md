# EvidenceLens — Full Project Plan

> **Source.** This file is the in-repo copy of the approved orchestrator plan. The canonical author-time copy lives at `C:\Users\TranJN\.claude\plans\evidencelens-orchestrator-soft-donut.md`. Both should stay in sync; the in-repo copy is what every contributor and downstream agent reads.

## Context

**What.** A free, public, agentic biomedical evidence search engine that unifies PubMed + preprints (bioRxiv/medRxiv) + clinical trials (CTG + ICTRP) + FDA/EMA regulatory + CMS Open Payments + NIH/NSF funding behind a hybrid (BM25 + vector + citation + recency) ranker, surfaces conflict-of-interest badges next to every author name, and lets visitors synthesize answers via three free inference tiers (BYOK, MCP, in-browser WebLLM).

**Why this plan exists.** The user is acting as orchestrator over a swarm of coding agents per the EvidenceLens orchestrator prompt. The complete spec ([../../moogle/docs/EVIDENCELENS_SPEC.md](../../moogle/docs/EVIDENCELENS_SPEC.md), 1745 lines) is the source of truth. Moogle is the architectural template for service skeletons, batching, and concurrency idioms; the spec wins where they conflict. This plan is the single approved blueprint for execution; per the user, I do most work directly and fan out sub-agents only for genuinely independent bursts (12 ingesters, scorer sub-services, frontend page surface).

**Confirmed decisions.**
- Greenfield repo at `c:\Trusted\evidencelens\`. `c:\Trusted\EvidenceLens0\` is ignored entirely (separate single-commit MVP-0; not reused, not deleted).
- Cost ceiling: ≤ $15/year recurring (domain only). All other infra free-tier on user's existing TrueNAS + Dokploy VPS + Tailscale + Cloudflare account.
- Dispatch posture: sequential streams; parallelism reserved for the 12 ingesters and the 4 scorer sub-services where work is genuinely independent.
- Full system from day one — no MVP phasing — per spec anti-goals.

**Scope-defining numbers from the spec.** 9 work streams (A–I), 12 ingesters, ~32 data sources catalog, 4 Cloudflare Workers, 5 NAS data services (Postgres, NATS JetStream, Meilisearch, Qdrant, Neo4j), 3 Dokploy services (gateway, agent, MCP), 1 Cloudflare Pages frontend, 1 GPU embedder, full OTel + Sentry + PostHog instrumentation, WCAG 2.2 AA. SLOs: search p95 ≤ 800ms, indexer lag ≤ 5min, recall fanout ≤ 1min, LCP ≤ 2.0s p75. Capacity floor: 5 QPS sustained / 50 burst / 500 docs-per-sec indexing.

---

## Repository bootstrap (Day 1)

Create `c:\Trusted\evidencelens\` as a fresh git repo. Layout exactly per spec §17:

```
evidencelens/
├── README.md, CLAUDE.md, LICENSE (MIT), .gitignore, .editorconfig
├── docs/{architecture.md, runbooks/, a11y/, api/, mcp/, sources/,
│        orchestrator-acknowledgement.md, orchestrator-plan.md,
│        status/, escalations/, rfcs/}
├── proto/{buf.yaml, buf.gen.yaml, evidencelens/v1/{document,events,scorer,embedder}.proto, gen/}
├── infra/{terraform/, docker-compose.yml, docker-compose.nas.yml, grafana/}
├── prompts/agent_system.md
├── config/{synonyms.json, stopwords.txt, source-rate-limits.yaml}
├── ingest/        # Go workspace (12 cmd/ + pkg/ingestcommon, pkg/otel, pkg/pubsubpub, pkg/r2, pkg/watermark)
├── index/         # Go workspace (cmd/indexer + pkg/batchers/{meili,qdrant,neo4j})
├── process/       # Python uv workspace (FastAPI processor + parsers/ + entity_linker, chunker, embedder_client, author_payment_joiner, publisher)
├── embedder/      # Python uv workspace (vLLM gRPC server)
├── scorer/        # Python uv workspace (gRPC: bm25, vector, citation, recency, fusion, ltr)
├── gateway/       # NestJS 11 (SearchModule, DocumentModule, TrialsModule, RecallsModule, LlmProxyModule, AdminModule)
├── agent/         # Python FastAPI BYOK proxy (providers/, tools.py)
├── mcp-server/    # TypeScript MCP v2025-06 (server.ts, tools.ts, resources.ts)
├── frontend/      # Next.js 15 App Router + shadcn/ui + TanStack Query + Zustand
├── workers/       # Cloudflare: pubsub-bridge/, click-logger/, webllm-shard/, turnstile-verify/
├── eval/          # queries.jsonl, run.py
├── tests/         # integration/, e2e/, load/
└── .github/workflows/
    ci.yml, deploy-{frontend,workers,cloud-run,nas}.yml,
    scheduled-{ingest,eval,pagerank,ltr-train,load-test}.yml
```

Workspace files (committed Day 1, even if empty): `pnpm-workspace.yaml`, `go.work`, `uv.lock`, `buf.yaml`. Pre-commit hook config (prettier+eslint, gofmt+golangci-lint, ruff+mypy, buf format+lint).

Initial commit: scaffold only, no logic. Push to fresh public repo `evidencelens/evidencelens` (orchestrator confirms repo creation with user before push).

---

## Reference patterns to lift from Moogle

| Pattern | Moogle source | EvidenceLens target |
|---|---|---|
| Go service skeleton (flags + signal + graceful shutdown) | [services/spider/cmd/spider/main.go:25-39](../../moogle/services/spider/cmd/spider/main.go#L25-L39) | `ingest/cmd/ingester-*/main.go`, `index/cmd/indexer/main.go` |
| Python service skeleton (signal + handle_exit) | [services/indexer/main.py:21-38](../../moogle/services/indexer/main.py#L21-L38) | `process/main.py`, `embedder/main.py`, `scorer/main.py`, `agent/main.py` |
| Env-var config helper `getEnv(key, fallback)` | [services/spider/cmd/spider/main.go:17-23](../../moogle/services/spider/cmd/spider/main.go#L17-L23) | `pkg/ingestcommon/env.go` (shared) |
| Batching with size threshold + signal flush | [services/indexer/main.py:81-108](../../moogle/services/indexer/main.py#L81-L108) | `index/pkg/batchers/{meili,qdrant,neo4j}.go` — extend with **time trigger** |
| Goroutine worker pool concurrency | [services/spider/internal/crawler/crawler.go:12-21](../../moogle/services/spider/internal/crawler/crawler.go#L12-L21) | All ingesters + indexer |
| Backpressure via queue size check | [services/spider/cmd/spider/main.go:69-92](../../moogle/services/spider/cmd/spider/main.go#L69-L92) | Processor (NATS lag check before pulling more from Pub/Sub) |
| Per-service `Dockerfile` + `docker-compose.yml` | every `services/*/` | every top-level service dir |
| GHA build-on-release matrix | [.github/workflows/build-docker-images.yml](../../moogle/.github/workflows/build-docker-images.yml) | extend per spec §16.2 to per-stream deploy workflows |

**Explicitly NOT copied from Moogle:** stub `monitoring/` Rust service, indexer placeholder CI tests, REST-only inter-service communication, no-OTel observability, sync-fetch frontend rendering, Python service file duplication.

---

## Open decisions (deferred to user / benchmark)

| Marker | Spec § | Default | Reconsider when |
|---|---|---|---|
| Text index engine | 19.1 | **Meilisearch** | p95 query > 200ms on full corpus. Fallbacks: Typesense, OpenSearch. |
| Vector engine | 19.2 | **Qdrant** | Filtered-search slow on faceted biomedical queries. Milvus consolidation acceptable. |
| Citation graph backend | 19.3 | **Neo4j Community** | PageRank on ~4B OpenAlex edges > 6h. Fallbacks: Apache AGE, Memgraph. |

**Risks I escalate immediately:** OpenAlex 300GB bulk, GCP Cloud Run free-tier exhaustion, Cloudflare Workers 100k req/day cap, Grafana Cloud 10k series cap, BigQuery 1TB query/mo, Open Payments fuzzy join false-positive rate.

---

## Sequenced execution

### Stream A — Platform & Infra (Week 1)

Provision free-tier accounts, buy domain, Terraform skeleton, sops + age secrets, NAS Docker stack (Postgres 16, NATS JetStream 2.11, Meilisearch 1.13, Qdrant 1.12, Neo4j 5 Community, OTel Collector, Prometheus, Loki agent), local dev compose, Cloudflare Pages stub, CI scaffold.

**Done when:** `terraform plan` clean; NAS compose all healthy; Cloudflare Pages serves placeholder over HTTPS at apex; CI green on no-op PR.

### Stream B — Shared Schemas & Interfaces (Week 1, **freeze EOW1**)

Author Protobuf (document, events, scorer, embedder), `buf generate` → `proto/gen/`, GraphQL SDL, OpenAPI 3.1, WebSocket message catalog, MCP tool catalog (8 tools), event schemas, `@evidencelens/contracts` workspace package.

**Done when:** `buf lint` clean; codegen works in Go + Python + TS; freeze tag `contracts-v1.0.0`. Post-freeze: any contract change requires `rfc-interface` PR.

### Stream C — Ingestion (Weeks 2–4)

Build `pkg/ingestcommon/` first (HTTP retry/backoff, watermark, R2 archival, Pub/Sub publisher, OTel, structured logging). Build `ingester-pubmed` as the template. Fan out 11 worker agents in batches for the rest. Open Payments gets manual extra care.

**Done when:** every ingester green CI + Cloud Run deploy + R2 raw tree populated + `RawDocEvent` flowing.

### Stream D — Processing & Embedding (Weeks 3–5)

Embedder (vLLM + BGE-M3, GPU, gRPC). Processor (FastAPI, 50 concurrent pipelines). Per-source parsers. Author × Open Payments joiner with `rapidfuzz` ≥ 90% threshold + 30d Postgres cache. Throughput target 500 docs/sec sustained.

**Done when:** smoke test ingests 1000 PubMed docs end-to-end with embeddings + COI populated, throughput ≥ 500/sec.

### Stream E — Indexing (Weeks 4–5)

Indexer with three batchers (Meili 1000/5s, Qdrant 100/5s, Neo4j 500). DLQ to `dlq.indexer` after 5 failures.

**Done when:** record ingester → indexer → searchable in all 3 indexes within 5min p95 over 24h.

### Stream F — Query, Ranking, Agent, MCP (Weeks 5–7)

F1 scorer-pool (4 sub-scorers + RRF + XGBoost LTR; **fan out 4 sub-agents** for the sub-scorers), F2 gateway (NestJS, integration spine, written by me), F3 agent (BYOK SSE proxy), F4 MCP server.

**Done when:** first wave ≤ 250ms p95, all waves ≤ 800ms p95; agent SSE works with real Anthropic key; MCP works in Claude Desktop.

### Stream G — Frontend (Weeks 6–8)

Next.js 15 + shadcn/ui + Tailwind 4. WebSocket-streamed result waves with ARIA live regions. COIBadge flagship component. BYOK key manager (localStorage only). WebLLM with manual tool-use loop. Lighthouse + axe-core gated in CI.

**Done when:** Lighthouse green, axe-core 0 violations, full search flow works, BYOK + WebLLM functional, recall ticker live.

### Stream H — Analytics & Experimentation (Weeks 7–8)

Click logger Worker → BigQuery. Firestore A/B assignment + Postgres audit. Pre-launch experiments configured. Interleaving evaluator. Required Grafana dashboards.

**Done when:** clicks flow frontend → BigQuery; A/B cookies stable; synthetic experiment shows audit rows.

### Stream I — Documentation (continuous, Weeks 1–8+)

Per-service READMEs, architecture doc, runbook per Grafana alert, auto-generated API docs, public docs site at `evidencelens.app/docs`, `BUILT-WITH.md`, launch posts.

---

## Quality gate (per stream)

Code merged. Unit coverage ≥ 80%. Integration tests pass in compose. E2E happy path. Public functions doc'd. Runbook present. OTel + metrics + logs in Grafana Cloud. Sentry breadcrumbs in user-facing flows. Service README per spec §18.4. Performance budget verified. Accessibility budget verified for frontend.

## Launch gate

All 9 streams done. Full corpus indexed. nDCG@10 ≥ 0.65. SLOs met for 7 consecutive days on staging. k6 full-cap load test passed. No P0/P1 issues. Launch posts reviewed. DNS pointed at production stack.

---

## Verification

```bash
# Local smoke (after every stream lands)
cd c:/Trusted/evidencelens
docker compose -f infra/docker-compose.yml up -d
make smoke
```

Nightly CI E2E. Eval drift check posts nDCG drift > 1% as PR comment. k6 nightly half-cap, weekly full-cap.

---

## Escalation triggers

I escalate to user (write `docs/escalations/{date}-{slug}.md`, pause stream) when: cost overrun unavoidable, spec inconsistency, `[ORCHESTRATOR DECISION]` needs benchmarking, source ToS changed, free-tier deprecated/capped below spec, irreversible action needed, fuzzy-join false-positive rate > 5%, WebLLM model > 2GB compressed.

---

## Anti-goals

- No MVP phasing.
- No user accounts / personalization.
- No paid services beyond domain.
- No mobile native app.
- No ship without WCAG 2.2 AA.
- No ship without OTel on every service.
- No spec-violating features without `RFC-feature` PR.
- No server-side LLM token burn.
