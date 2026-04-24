# Experiment: rrf_k_value

**Status:** Pre-launch (defined, not yet enrolling).
**Owner:** scorer-pool team.
**Spec ref:** §11.5.

## Hypothesis

Lower `k` in Reciprocal Rank Fusion (default 60 → candidate 30) more
aggressively rewards top-of-list items from each sub-scorer. This may
improve the share of clicks landing in the top-3 results when the user
typed a high-precision query (e.g. a specific drug + condition).

## Variants

| Variant | Weight | Params |
|---|---|---|
| control    | 50% | `{ k: 60 }` |
| aggressive | 50% | `{ k: 30 }` |

## Primary metric

`ndcg_at_10` — measured nightly via `eval/run.py` against the staging
gateway with the variant header applied. Significance test: paired
t-test from `eval/interleaving.py` over 7 days.

## Stop conditions

- nDCG@10 of `aggressive` regresses ≥ 1% vs control for 3 consecutive nights.
- Per-query latency p95 increases > 50ms (RRF is O(N), so unlikely).

## Promotion criteria

`aggressive` ndcg_at_10 ≥ control + 0.5 percentage points with paired
t-test p < 0.05 over ≥ 5,000 sessions.

## Bucketing

Frontend reads `GET /api/experiments/assignment?session_id=…&keys=rrf_k_value`
and stamps the variant on every search request. SHA-256(session+key) %
10000 bucketing in `gateway/src/experiments/experiments.module.ts`.
