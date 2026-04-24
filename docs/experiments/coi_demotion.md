# Experiment: coi_demotion

**Status:** Pre-launch (highest-stakes experiment of the four).
**Owner:** scorer-pool team + product.
**Spec ref:** §11.5.

## Hypothesis

Result authors with > $50k in same-year Open Payments matches may
introduce systematic bias into the visible top results. Lightly
demoting them (multiplicative 0.7 on the LTR score) may improve the
overall evidence quality without destroying recall.

This is a deliberate value-laden experiment; the team must approve
launch and a public methodology page is required if it's promoted.

## Variants

| Variant | Weight | Params |
|---|---|---|
| control | 50% | `{ coi_demotion: 0.0 }` (no demotion) |
| demote  | 50% | `{ coi_demotion: 0.3 }` (multiply LTR score by 0.7 when has_coi_authors=true and max_author_payment_usd>50000) |

## Primary metric

`dwell_time_mean` — average seconds spent on the document drawer for
top-5 clicked results. Proxy for "did the user find this useful?"

## Stop conditions

- Click rate falls > 8% (users may not see the most-discussed papers).
- Eval nDCG@10 against the manually-judged set falls > 3% (we'd be
  systematically demoting genuinely-relevant landmark trials).
- Public criticism that the demotion is opaque or unfair.

## Promotion

Requires:
1. Dwell-time improvement ≥ 10% with paired t-test p < 0.05.
2. nDCG@10 unchanged or better.
3. Public methodology page on `/about/coi-demotion` published with the
   exact threshold and rationale.
4. Sign-off from at least one external reviewer with biomedical
   expertise.
