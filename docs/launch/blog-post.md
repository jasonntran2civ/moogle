# EvidenceLens — a free, agentic biomedical evidence search with COI badges

*Draft launch blog post. Aimed at r/medicine, r/MachineLearning, Stat News, MedCity News, Hacker News.*

## Why I built this

Every time I needed to look up the evidence on a treatment, I ended up in five tabs: PubMed for the trials, ClinicalTrials.gov for what's recruiting, openFDA for approvals and recalls, OpenAlex for the citation graph, and (last and most awkwardly) the CMS Open Payments search tool to figure out whether the doctors who wrote the studies had been receiving large checks from the drug's manufacturer.

That last step matters enormously, and there's no good tool for it.

EvidenceLens stitches those sources together into one search engine, surfaces a **COI badge next to every author name**, and lets you ask follow-up questions to an LLM **without paying for any LLM costs** (you bring your own key, or your MCP client pays, or the model runs in your browser).

## What's under the hood

A polyglot pipeline:

1. **Twelve Go ingesters** on Google Cloud Run pull from PubMed, bioRxiv/medRxiv, ClinicalTrials.gov, WHO ICTRP, openFDA (drug approvals, recalls, MAUDE, 510(k)), OpenAlex bulk + REST, CrossRef, Unpaywall, NIH RePORTER, CMS Open Payments, Cochrane, and USPSTF/NICE/AHRQ guidelines.
2. **A Python processor** on a TrueNAS NAS consumes raw events from Pub/Sub, parses each source's native format into a canonical Protobuf `Document`, runs entity linking (scispaCy + UMLS), chunks for embedding, embeds via gRPC against a vLLM-served BGE-M3 model on the GPU, joins author records against Open Payments via fuzzy match (rapidfuzz, ≥0.90 confidence), and publishes to NATS JetStream.
3. **A Go indexer** writes to Meilisearch (text + facets), Qdrant (1024-d HNSW vectors), and Neo4j Community (citation graph with payment edges), with per-target batchers and a dead-letter queue for failures.
4. **A Python scorer pool** runs four sub-scorers in parallel — BM25 over Meilisearch, vector cosine over Qdrant, citation PageRank over Neo4j, exponential recency decay — fuses them with Reciprocal Rank Fusion (k=60), then reranks the top 50 with an XGBoost LambdaMART head trained on synthetic relevance labels (replaced by click-trained data after launch).
5. **A NestJS gateway** exposes REST + GraphQL + WebSocket + a BYOK LLM proxy. Result waves stream over WebSockets so the first results appear within ~250ms.
6. **A Python agent service** acts as the BYOK proxy — Anthropic with prompt caching, OpenAI/Groq/OpenRouter/Together/DeepInfra via the same shape, and a fallback to your own local Ollama. **Keys are never stored.**
7. **A TypeScript MCP server** exposes 8 evidence tools to Claude Desktop, Cursor, Cline, and Goose users.
8. **A Next.js 15 frontend** on Cloudflare Pages with a hard WCAG 2.2 AA bar, Lighthouse-gated performance budgets, and an in-browser WebLLM tier (Llama 3.2 3B) for users who want zero server dependency.

## Why $0/yr

Every choice was made under a recurring-cost ceiling of zero dollars per year. The maintainer already pays for a TrueNAS NAS and a small Dokploy VPS — those run the heavy long-lived services. Everything else fits inside free tiers:

- Cloudflare Pages + Workers + R2 ($0 egress!) + KV + Tunnel + Turnstile
- GCP Cloud Run + Pub/Sub + BigQuery + Firestore
- Grafana Cloud (Tempo + Loki + Mimir)
- Sentry + PostHog
- GitHub Actions (unlimited for public repos)
- Oracle Cloud Always Free for failover

There's no domain — we live at `evidencelens.pages.dev`, and the MCP server at `mcp-evidencelens.<account>.workers.dev`. Total recurring: **$0**.

## What it isn't

- **Not medical advice.** The disclaimer is on every page.
- Not a replacement for clinical decision support tools.
- Not a substitute for reading the primary sources.
- Not a personalization engine — we don't track users or build profiles.

## What's next

It's open source (MIT). Source: https://github.com/evidencelens/evidencelens. Issues, PRs, and especially **feedback on the COI matching policy** (the highest-stakes piece) are very welcome.

If you find it useful and you can afford to, give the upstream OSS dependencies stars or money — see [BUILT-WITH.md](../../BUILT-WITH.md).
