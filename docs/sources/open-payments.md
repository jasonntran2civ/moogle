# CMS Open Payments — flagship feature

| Field | Value |
|---|---|
| Tier | Conflict-of-interest |
| Records | ~14M payments/year × 7+ years available |
| Access | Annual bulk CSV from `download.cms.gov/openpayments/PGYY_P0NNNNNN.zip` |
| License | Public domain |
| Refresh | Annual bulk + monthly check |
| Implementation | [ingest/cmd/ingester-open-payments/](../../ingest/cmd/ingester-open-payments/) |

## Why this matters

Every result in EvidenceLens shows a **COI badge** next to author names. Open Payments is the data behind those badges — pharma → physician payments by sponsor, year, type, and amount. This is the project's flagship differentiator.

## Implementation

Two roles in one service:

1. **Bulk ingest** (`POST /run`): downloads annual CSVs, COPY-loads into Postgres `open_payments` table.
2. **Lookup endpoint** (`GET /lookup?name=...&state=...&year=...`): synchronous HTTP API consumed by the processor's `author-payment-joiner`.

## Matching policy (CRITICAL)

Author names in literature are typically `Lastname FM` (initials only). CMS uses full names + NPI. Matching uses Postgres `pg_trgm` similarity over `physician_name` with a **conservative threshold of 0.90** (configurable via `MIN_FUZZY_CONFIDENCE` env).

**Bias is conservative**: false positives (incorrectly attributing payments to an author who didn't receive them) are *much worse* than false negatives (missing some real matches). When in doubt, return nothing.

When state is provided in the query (e.g. from author affiliation), restrict to `physician_state = state` — this dramatically reduces false-positive risk.

If the processor ever observes >5% false-positive rate in spot-checks, that's an escalation trigger per [orchestrator-plan.md](../orchestrator-plan.md) "Escalation triggers".

## Cache

Results are cached in Postgres `author_payment_cache` for 30 days, keyed by normalized `lastname:firstname:state`. The cache TTL bounds the lag between a new annual Open Payments release and matched records.
