# Show HN draft

**Title:** Show HN: EvidenceLens – free, agentic biomedical evidence search with COI badges

**Body:**

Hi HN,

I built EvidenceLens because every time I tried to look up the evidence on a drug or a treatment I had to bounce between PubMed, ClinicalTrials.gov, openFDA, and (separately) the CMS Open Payments database to figure out whether the doctors who wrote the studies were on the manufacturer's payroll. That last step was the killer — it's *extremely* important context and there's no good tool for it.

EvidenceLens unifies all of those:

- **Sources**: PubMed (~38M), bioRxiv/medRxiv preprints, ClinicalTrials.gov + WHO ICTRP, openFDA approvals + recalls + adverse events, OpenAlex (~250M works + ~4B citation edges), CrossRef + Unpaywall enrichment, NIH RePORTER funding, Cochrane systematic reviews, USPSTF/NICE/AHRQ guidelines, and the **CMS Open Payments** records that drive the COI badges.
- **Ranking**: hybrid BM25 (Meilisearch) + vector (Qdrant + BGE-M3) + citation PageRank (Neo4j) + recency, fused with Reciprocal Rank Fusion, then reranked by an XGBoost LambdaMART head.
- **Conflict-of-interest badges** next to author names — fuzzy-matched against Open Payments with a conservative ≥0.90 confidence threshold.
- **Three free inference tiers**: Bring Your Own Key, MCP (use it from Claude Desktop / Cursor / Cline), or in-browser WebLLM (Llama 3.2 3B). EvidenceLens itself never burns server-side LLM tokens.
- **Live FDA recall fanout** over WebSocket — when a recall hits openFDA, subscribed clients see it within ~1 minute.

It's free and it stays free: total recurring cost is **$0/year**. Everything runs on Cloudflare Pages / Workers / R2, GCP free tier (Cloud Run + Pub/Sub + BigQuery + Firestore), Grafana Cloud, and my own NAS + small VPS.

Stack is a polyglot monorepo: 12 Go ingesters on Cloud Run, Python processor + vLLM-backed embedder + scorer on the NAS, Go indexer with three batchers (Meili/Qdrant/Neo4j), NestJS gateway, Python BYOK agent proxy, TypeScript MCP server, Next.js 15 frontend with WCAG 2.2 AA + Lighthouse-gated perf.

Not medical advice. Always verify against primary sources. The COI badges use fuzzy matching against public records and may have false positives — the link below the badge takes you straight to the CMS source record so you can check.

Repo: https://github.com/evidencelens/evidencelens
Live: https://evidencelens.pages.dev

Happy to discuss the ranking pipeline, the COI fuzzy-match policy, or how to keep a project like this at $0/yr.
