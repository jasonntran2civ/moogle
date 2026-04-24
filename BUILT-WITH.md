# Built with — and thanks to

EvidenceLens is the work of a small open-source community standing on a *very* long shoulder of giants. This file lists every free-tier service and OSS dependency the project relies on, with thanks. If you make use of this project, consider giving these folks money or stars.

## Free-tier services

- **[Cloudflare](https://www.cloudflare.com/)** — Pages, Workers, R2 ($0 egress!), KV, Tunnel, Turnstile, Registrar.
- **[Google Cloud](https://cloud.google.com/free)** — Cloud Run, Pub/Sub, BigQuery, Firestore.
- **[Grafana Cloud](https://grafana.com/products/cloud/)** — Tempo, Loki, Mimir.
- **[Sentry](https://sentry.io/)** — frontend error reporting.
- **[PostHog](https://posthog.com/)** — product analytics (self-host fallback also available).
- **[GitHub Actions](https://github.com/features/actions)** — unlimited minutes for public repos.
- **[Oracle Cloud Always Free](https://www.oracle.com/cloud/free/)** — gateway failover (4 ARM vCPU + 24GB RAM).
- **[Tailscale](https://tailscale.com/)** — secure mesh between TrueNAS, Dokploy, Oracle, dev machines.

## Public biomedical data sources

PubMed, PubMed Central, bioRxiv, medRxiv, ClinicalTrials.gov, WHO ICTRP, openFDA, EMA OpenData, OpenAlex, CrossRef, Unpaywall, NIH RePORTER, NSF Awards, CMS Open Payments, Cochrane Library (metadata only), USPSTF, NICE, AHRQ, plus reference sources — see [docs/sources/](docs/sources/) for per-source attribution and license terms.

## OSS dependencies (selected)

### Languages + runtimes
[Go](https://go.dev/), [Python](https://www.python.org/), [TypeScript](https://www.typescriptlang.org/), [Node.js](https://nodejs.org/).

### Frameworks
- Go: [pgx](https://github.com/jackc/pgx), [nats.go](https://github.com/nats-io/nats.go), [neo4j-go-driver](https://github.com/neo4j/neo4j-go-driver), [meilisearch-go](https://github.com/meilisearch/meilisearch-go), [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2), [backoff](https://github.com/cenkalti/backoff), [Colly](https://go-colly.org/).
- Python: [FastAPI](https://fastapi.tiangolo.com/), [vLLM](https://github.com/vllm-project/vllm), [sentence-transformers](https://www.sbert.net/), [scispaCy](https://github.com/allenai/scispacy), [rapidfuzz](https://github.com/rapidfuzz/RapidFuzz), [tiktoken](https://github.com/openai/tiktoken), [XGBoost](https://xgboost.readthedocs.io/), [structlog](https://www.structlog.org/), [httpx](https://www.python-httpx.org/), [asyncpg](https://magicstack.github.io/asyncpg/).
- TypeScript: [Next.js](https://nextjs.org/), [React](https://react.dev/), [NestJS](https://nestjs.com/), [Apollo](https://www.apollographql.com/), [Tailwind CSS](https://tailwindcss.com/), [shadcn/ui](https://ui.shadcn.com/), [TanStack Query](https://tanstack.com/query), [Zustand](https://github.com/pmndrs/zustand), [WebLLM](https://webllm.mlc.ai/), [@modelcontextprotocol/sdk](https://github.com/modelcontextprotocol/typescript-sdk).

### Data plane
[Meilisearch](https://www.meilisearch.com/), [Qdrant](https://qdrant.tech/), [Neo4j Community](https://neo4j.com/), [PostgreSQL](https://www.postgresql.org/), [NATS JetStream](https://nats.io/), [Redis](https://redis.io/).

### ML models
- **[BAAI/bge-m3](https://huggingface.co/BAAI/bge-m3)** — primary embedding model (1024-d).
- **[BAAI/bge-small-en-v1.5](https://huggingface.co/BAAI/bge-small-en-v1.5)** — CPU fallback (384-d).
- **[Llama 3.2 3B](https://huggingface.co/meta-llama/Llama-3.2-3B)** — WebLLM in-browser.

### Tooling
[buf](https://buf.build/), [pnpm](https://pnpm.io/), [uv](https://github.com/astral-sh/uv), [Terraform](https://www.terraform.io/), [Docker](https://www.docker.com/), [k6](https://k6.io/), [Lighthouse CI](https://github.com/GoogleChrome/lighthouse-ci), [axe-core](https://github.com/dequelabs/axe-core), [Playwright](https://playwright.dev/), [sops](https://github.com/getsops/sops), [age](https://github.com/FiloSottile/age).

### Reference

The architecture is patterned after [Moogle](https://github.com/IonelPopJara/search-engine), an educational polyglot search engine. Many service skeleton patterns (env-var config, signal handling, batching, concurrency) come from there. Thanks to the Moogle authors for showing the shape.

---

If you contribute to anything on this list, please open an issue or PR — we'd love to add a personal thanks.
