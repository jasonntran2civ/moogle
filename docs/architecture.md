# Architecture

EvidenceLens is a polyglot monorepo of independently deployable services. This doc is the high-level map; per-service details live in each service's `README.md`.

## Diagram

```
┌─ ingest (Go, GCP Cloud Run, 12 ingesters) ─────────────────────────────────┐
│   pubmed, preprint, trials, ictrp, fda, openalex, crossref,                │
│   unpaywall, nih-reporter, open-payments, cochrane, guidelines             │
└────────────┬───────────────────────────────────────────────────────────────┘
             │  raw bytes ────────────► R2  raw/{source}/{date}/{id}.json.gz
             │  RawDocEvent ──► Pub/Sub raw-docs ──► pubsub-bridge Worker
             │                                            │
             ▼                                            ▼
                                                  NATS JetStream
                                                  (raw-docs.{source})
                                                            │
                                                            ▼
              ┌─ process (Python, NAS Docker) ──────────────────────────────┐
              │  parse → normalize → entity-link → chunk → embed →           │
              │  Author×OpenPayments fuzzy join (≥0.90 confidence) → publish │
              └────────────┬─────────────────────────────────────────────────┘
                           │ embed() gRPC
                           ▼
              ┌─ embedder (Python + vLLM, NAS GPU, gRPC) ────────────────────┐
              │  BGE-M3 1024-d (GPU) | BGE-small 384-d (CPU fallback)        │
              └────────────────────────────────────────────────────────────────┘
                           │
                           ▼ NATS indexable-docs.{source}
              ┌─ indexer (Go, NAS Docker) ───────────────────────────────────┐
              │  meili (1000/5s) | qdrant (100/5s) | neo4j MERGE (500)       │
              │  DLQ → NATS dlq.indexer after 5 failures                     │
              └────────────┬─────────────────────────────────────────────────┘
                           ▼
              ┌─ data plane ─────────────────────────────────────────────────┐
              │  Meilisearch (text+facets) | Qdrant (1024-d HNSW) |          │
              │  Neo4j Community (citation graph) | Postgres (op state)      │
              └────────────┬─────────────────────────────────────────────────┘
                           ▼
              ┌─ scorer-pool (Python gRPC) ──────────────────────────────────┐
              │  bm25 + vector + citation + recency → RRF k=60 → XGBoost LTR │
              │  Streams 3 waves @ 200/500/1000 ms                           │
              └────────────┬─────────────────────────────────────────────────┘
                           ▼ gRPC
              ┌─ gateway (NestJS, Dokploy VPS) ─────────────────────────────┐
              │  REST /api/* | GraphQL /graphql | WS /ws | BYOK proxy /llm/* │
              └────┬─────────────┬──────────────────────────────────────────┘
                   │             │
                   │             └─► agent-service (Python BYOK SSE)
                   │                  ─► Anthropic / OpenAI-compat / Ollama
                   ▼
              ┌─ frontend (Next.js 15, Cloudflare Pages) ────────────────────┐
              │  /, /search, /document/[id], /trial/[id], /recalls, /docs    │
              │  WebSocket-streamed result waves with COIBadges              │
              │  3-tier inference: BYOK | MCP | WebLLM (in-browser)          │
              └────────────────────────────────────────────────────────────────┘

mcp-server (TypeScript, Dokploy VPS, stdio + http+sse)
  └── 8 MCP tools dispatch to gateway POST /api/tool/{name}

workers (Cloudflare): pubsub-bridge | click-logger | webllm-shard | turnstile-verify

analytics: clicks → BigQuery analytics.clicks (partitioned daily, clustered by variant+query_text)
A/B: Firestore experiments/{key} + ab_assignments/{session}/{key}; audit in Postgres ab_assignment_audit

observability: every service → OTel Collector (NAS) → Grafana Cloud (Tempo + Loki + Mimir)
                frontend → Sentry breadcrumbs + PostHog product analytics
```

## Why this shape

- **Polyglot intentional**: Go for ingesters and indexer (concurrency, small Cloud Run images); Python for processor/embedder/scorer (ML/NLP ecosystem); TypeScript for gateway/MCP/frontend (type-safe public API surface).
- **Free-tier first**: GCP Cloud Run + Pub/Sub + BigQuery + Firestore for the elastic ingest fan-in; Cloudflare Pages + Workers + R2 for the user-facing edge; TrueNAS for the heavy long-running data plane (Meilisearch + Qdrant + Neo4j + Postgres + NATS); Dokploy VPS for stateful long-lived gateway/agent/MCP that need persistent connections.
- **One-way data flow**: ingesters never read from indexes; indexer never reads from Pub/Sub. Eliminates entire classes of bug.

## SLOs (spec §14.1)

See [infra/grafana/alerts/slo.yaml](../infra/grafana/alerts/slo.yaml) for the alert definitions.

## Cost ceiling

**$0/year recurring.** No domain (free `evidencelens.pages.dev` subdomain). All compute on free tiers + the maintainer's existing TrueNAS + Dokploy VPS hardware.
