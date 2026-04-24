# EvidenceLens

A free, public, agentic biomedical evidence search engine.

EvidenceLens unifies PubMed, preprints (bioRxiv / medRxiv), clinical trials (ClinicalTrials.gov + WHO ICTRP), FDA / EMA regulatory data, conflict-of-interest records (CMS Open Payments), and funding sources (NIH RePORTER, NSF Awards) behind a hybrid (BM25 + vector + citation + recency) ranker. Every result surfaces conflict-of-interest badges next to author names. Visitors synthesize answers via three free inference tiers — Bring-Your-Own-Key, Model Context Protocol, or in-browser WebLLM — so EvidenceLens never burns server-side LLM tokens.

**Status:** Greenfield, in active build. Architecture per [docs/architecture.md](docs/architecture.md). Source-of-truth spec at [`EVIDENCELENS_SPEC.md`](../moogle/docs/EVIDENCELENS_SPEC.md).

## Quick start

```bash
cp infra/.env.example infra/.env       # fill in required env vars (see infra/README.md)
docker compose -f infra/docker-compose.yml up -d
make smoke                              # end-to-end smoke test
```

Frontend dev: `pnpm --filter frontend dev`. Gateway dev: `pnpm --filter gateway start:dev`. See per-service `README.md` for individual workflows.

## Architecture (one paragraph)

A polyglot pipeline: 12 Go ingesters on GCP Cloud Run pull from public biomedical APIs, archive raw responses to Cloudflare R2, and publish `RawDocEvent` to GCP Pub/Sub. A Python processor on TrueNAS consumes via a `pubsub-bridge` Cloudflare Worker, parses + normalizes + entity-links + chunks + embeds (gRPC to a vLLM-backed BGE-M3 service on a TrueNAS GPU) + joins Open Payments by fuzzy name match, then publishes `IndexableDocEvent` to NATS JetStream. A Go indexer batches into Meilisearch (text + facets), Qdrant (1024-d vectors), and Neo4j (citation graph). A Python gRPC scorer-pool runs BM25 + vector + citation + recency sub-scorers in parallel, fuses with Reciprocal Rank Fusion, and reranks with an XGBoost LambdaMART head. A NestJS gateway on Dokploy fans out to scorers and streams result waves over WebSockets to a Next.js frontend on Cloudflare Pages. An MCP server exposes the same tools to Claude Desktop / Cursor / Cline / Goose users.

See [docs/architecture.md](docs/architecture.md) for the full diagram.

## Cost

Total recurring: **≤ $15/year** (domain only). Everything else runs on free tiers (Cloudflare, GCP, Grafana Cloud, Sentry, PostHog, GitHub Actions) plus the maintainer's existing TrueNAS NAS and Dokploy VPS. See [BUILT-WITH.md](BUILT-WITH.md) for a complete list with thanks.

## License

MIT. See [LICENSE](LICENSE). EvidenceLens consumes data from sources with their own terms — we never serve full text from non-OA sources, only metadata and deep links. See [docs/sources/](docs/sources/) for per-source attribution.

## Disclaimer

EvidenceLens is a research tool. It is **not** medical advice, **not** clinical decision support, and **not** a substitute for a licensed clinician or pharmacist. Conflict-of-interest badges are computed from public records via fuzzy matching and may contain false positives or omissions. Always verify critical findings against primary sources.
