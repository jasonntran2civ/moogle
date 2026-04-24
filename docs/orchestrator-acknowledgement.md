# Orchestrator Acknowledgement

**Date:** 2026-04-24
**Plan file:** [orchestrator-plan.md](orchestrator-plan.md)
**Approval status:** Plan approved by user. Execution begun this date.

## Mission, in my own words

EvidenceLens is a free, public, agentic biomedical evidence search engine. It unifies twelve public biomedical data sources behind a hybrid (BM25 + vector + citation + recency) ranker reranked by a learned-to-rank head, surfaces conflict-of-interest badges next to every author name, and lets visitors synthesize answers via three free inference tiers (Bring-Your-Own-Key, Model Context Protocol, in-browser WebLLM) so EvidenceLens itself never burns server-side LLM tokens. The system is built as a polyglot monorepo (Go ingesters, Python processor / embedder / scorer / agent, NestJS gateway, TypeScript MCP server, Next.js frontend, Cloudflare Workers, Terraform infra) and runs entirely on free-tier cloud (Cloudflare, GCP, Grafana Cloud, Sentry, PostHog, GitHub Actions) plus the maintainer's existing TrueNAS NAS and Dokploy VPS. Recurring cost ≤ $15/year (domain only).

## My role

Lead orchestrator. I work directly on most streams; I dispatch specialized sub-agents only for the 12 ingesters and the 4 scorer sub-services where work is genuinely independent. I freeze contracts (proto, GraphQL, OpenAPI, MCP catalog) by end of week 1 and reject post-freeze changes without an `rfc-interface` PR. I escalate to the user when the spec is internally inconsistent, when a `[ORCHESTRATOR DECISION]` requires real-world benchmarking I cannot run, when a free tier's cap has changed below spec assumptions, or when an action is irreversible.

## Source-of-truth references

- [EvidenceLens spec](../../moogle/docs/EVIDENCELENS_SPEC.md) — 1745 lines, the contract for what gets built.
- [Moogle reference repo](../../moogle/) — architectural template for service skeletons, env-var config, batching, and concurrency. EvidenceLens diverges where the spec demands (gRPC + Protobuf, OTel, WebSocket-streamed result waves, NATS).
- [Approved plan](orchestrator-plan.md) — sequenced execution, per-stream details, quality gates, escalation triggers.

## Anti-goals (carried verbatim)

- No MVP phasing — full system from day one.
- No user accounts, authentication, or personalization.
- No paid services beyond domain.
- No mobile native app.
- No ship without WCAG 2.2 AA.
- No ship without OTel coverage on every service.
- No features outside spec without an `RFC-feature` PR.
- No server-side LLM token burn.

Acknowledged. Building.
