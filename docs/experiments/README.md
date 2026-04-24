# Experiments

Per-experiment documentation per spec §11.5. Each `.md` file describes
one named experiment registered in `config/experiments.yaml` and
served via `GET /api/experiments/assignment` (gateway).

## Pre-launch experiments

| Key | File | Goal |
|---|---|---|
| `rrf_k_value`         | [rrf_k_value.md](rrf_k_value.md)         | RRF k=30 vs k=60 |
| `recency_half_life`   | [recency_half_life.md](recency_half_life.md)   | 365 / 730 / 1460 day decay |
| `vector_weight_in_rrf`| [vector_weight_in_rrf.md](vector_weight_in_rrf.md)| Vector-heavy weighted RRF |
| `coi_demotion`        | [coi_demotion.md](coi_demotion.md)        | Demote high-COI papers |

## Workflow

1. Define hypothesis + variants + stop conditions in a new `.md` here.
2. Add the entry to `config/experiments.yaml` with `enabled: false`.
3. Open a PR labeled `rfc-experiment`. Reviewer checks the docs +
   primary metric + ethics (especially for value-laden experiments
   like `coi_demotion`).
4. Flip `enabled: true`, deploy. Frontend starts seeing variants
   on next page load.
5. Nightly `eval/interleaving.py` job + 7-day BigQuery aggregation
   produces a verdict.
6. Promote (flip control's params to the winner) or roll back.
