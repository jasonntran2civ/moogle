# Experiment: vector_weight_in_rrf

**Status:** Pre-launch.
**Owner:** scorer-pool team.
**Spec ref:** §11.5.

## Hypothesis

Pure RRF treats every sub-scorer equally. Weighted RRF that biases
toward semantic similarity (vector) may surface more topically relevant
papers for natural-language queries vs strict-keyword queries.

## Variants

| Variant       | Weight | Params |
|---|---|---|
| control       | 50% | `{ vector_weight: 1.0 }` (pure RRF) |
| vector_heavy  | 50% | `{ vector_weight: 1.5 }` |

## Primary metric

`click_position_mean` — average rank of the clicked result. Lower is
better; if vector-heavy makes clicked items appear higher, mean drops.

## Stop conditions

- Click rate (clicks/session) drops > 5%.
- BM25-only-precision queries (matched by `query_has_drug_entity`
  feature) regress in nDCG > 2%.

## Promotion

Mean clicked position improvement ≥ 0.5 ranks with paired t-test
p < 0.05.
