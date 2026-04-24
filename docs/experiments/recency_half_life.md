# Experiment: recency_half_life

**Status:** Pre-launch.
**Owner:** scorer-pool team.
**Spec ref:** §11.5.

## Hypothesis

Default half-life of 730 days (~2y) may over-weight stale evidence in
fast-moving fields (e.g. mRNA platform, oncology approvals). A 365-day
candidate boosts recency; a 1460-day candidate confirms behavior in
slow-moving fields (rare disease, longitudinal cohort). Three-way A/B
to triangulate.

## Variants

| Variant | Weight | Params |
|---|---|---|
| control | 33% | `{ half_life_days: 730 }` |
| short   | 33% | `{ half_life_days: 365 }` |
| long    | 34% | `{ half_life_days: 1460 }` |

## Primary metric

`ndcg_at_10` against the manually-judged eval set (eval/queries.jsonl)
plus query-level breakdown by intent (`clinical`, `oncology`, `safety`).

## Stop conditions

- Any variant shows nDCG drop > 2% on the `clinical` query subset.
- Result-set composition skews to documents from a single year (>40% of
  top-10 in one year).

## Promotion

Whichever variant maximizes nDCG aggregate AND doesn't degrade either
query subset by > 1%.
