# Launch gate

Project ships when **every** box is checked. No exceptions; no degraded launches.

## Code & tests

- [ ] All 9 streams complete per quality gates in [orchestrator-plan.md](../orchestrator-plan.md).
- [ ] CI green on `main` for 3 consecutive runs.
- [ ] Unit-test coverage ≥ 80% across all services.
- [ ] Integration tests pass against running docker-compose.
- [ ] E2E happy path passes (frontend → gateway → scorer → indexes → result with COI badge).
- [ ] No P0 / P1 issues open.

## Data

- [ ] Full corpus indexed per spec §2 catalog (or escalation filed for any non-viable source).
- [ ] Author × Open Payments fuzzy match spot-checked on 100 samples; false-positive rate < 5% (escalation trigger if not).
- [ ] Citation PageRank refresh completes in < 6h on full OpenAlex graph (escalation trigger for spec §19.3).

## Quality

- [ ] nDCG@10 ≥ 0.65 average across `eval/queries.jsonl` queries.
- [ ] Per-query nDCG ≥ each query's `min_ndcg`.
- [ ] LTR head trained on at least the synthetic-label set; click-trained model deferred until 30d post-launch.

## SLOs (spec §14.1) met for 7 consecutive days on staging

- [ ] Search availability ≥ 99.5%
- [ ] Search p95 ≤ 800ms
- [ ] First wave ≤ 250ms p95
- [ ] Indexer lag ≤ 5 min p95
- [ ] Recall fanout ≤ 1 min p95
- [ ] Frontend LCP ≤ 2.0s p75
- [ ] Frontend INP ≤ 200ms p75

## Load test

- [ ] [k6 load test](../../tests/load/search.js) at full spec §15.1 capacity (5 QPS sustained, 50 burst, 200 WS concurrent, 1000 peak) passes thresholds.

## Accessibility

- [ ] axe-core zero violations across all routes (`tests/a11y/axe.spec.ts`).
- [ ] Manual screen-reader smoke (NVDA + VoiceOver) on home, /search, /document/[id].
- [ ] Lighthouse CI green with budgets in [`frontend/.lighthouserc.json`](../../frontend/.lighthouserc.json).

## Security

- [ ] No secrets in repo (CI guardrails verify on every PR).
- [ ] CSP headers verified per `next.config.js`.
- [ ] BYOK proxy: keys never logged (manual audit of `byok_proxy_telemetry` schema + `agent/main.py`).
- [ ] Cloudflare Turnstile required for unauthenticated `/llm/*` requests.

## Comms

- [ ] [Show HN](show-hn.md) draft user-reviewed.
- [ ] [Launch blog post](blog-post.md) user-reviewed.
- [ ] DNS pointed at production stack (`evidencelens.pages.dev` confirmed live).
- [ ] Cloudflare Tunnel green; gateway reachable from public WSS.
- [ ] First-day on-call schedule shared.

## Rollback plan

- [ ] Documented rollback for each deploy workflow (Pages, Workers, Cloud Run, NAS) in [docs/runbooks/rollback.md](../runbooks/rollback.md).
- [ ] Last-known-good Docker image SHAs pinned for each NAS service.
- [ ] BigQuery export of `analytics.clicks` schedule confirmed running (so even if Firestore A/B is reset, click history survives).

## Day-of operations

- [ ] One person on-call for first 24h.
- [ ] Slack/Discord channel ready for community feedback.
- [ ] Status page (`evidencelens.pages.dev/status` if/when added) prepared.
- [ ] Bug tracker triage SLA agreed (initial response ≤ 24h for first week).
