# ingester-pubmed

PubMed ingester (spec §5.1.1). Reference implementation — every other ingester mirrors this shape.

## Source

[NCBI E-utilities](https://www.ncbi.nlm.nih.gov/books/NBK25497/) (`esearch` + `efetch`). Free, public. API key recommended (raises rate limit from 3/s to 10/s, free at <https://www.ncbi.nlm.nih.gov/account/settings/>).

## Watermark

PubMed `EDAT` (entry date), ISO-8601 string `YYYY/MM/DD`. Stored in Postgres `ingestion_state.last_high_watermark`.

## Run

```bash
# Local
DATABASE_URL=postgres://evidencelens:changeme-dev-only@localhost:5432/evidencelens \
R2_ACCOUNT_ID=... R2_ACCESS_KEY_ID=... R2_SECRET_ACCESS_KEY=... R2_BUCKET=evidencelens-raw \
R2_ENDPOINT=https://...r2.cloudflarestorage.com \
GCP_PROJECT=evidencelens-prod \
NCBI_API_KEY=... NCBI_EMAIL=you@example.com \
go run ./cmd/ingester-pubmed
# then POST to /run:
curl -X POST http://localhost:8080/run
```

## Deploy

Auto-deployed by `.github/workflows/deploy-cloud-run.yml` on push to `main`. Cloud Scheduler invokes `POST /run` every 6 hours per `infra/terraform/modules/gcp/main.tf` schedule.

## Env vars

| Var | Default | Notes |
|---|---|---|
| `DATABASE_URL` | required | Postgres DSN for ingestion_state + watermark |
| `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`, `R2_ENDPOINT` | required | R2 raw archive |
| `GCP_PROJECT` | required | Pub/Sub topic namespace |
| `PUBSUB_TOPIC_RAW_DOCS` | `raw-docs` | Pub/Sub topic to publish into |
| `NCBI_API_KEY` | empty | Recommended; raises rate limit |
| `NCBI_TOOL` | `evidencelens` | Tool identifier sent to NCBI |
| `NCBI_EMAIL` | `contact@example.com` | Required by NCBI ToS |
| `PUBMED_MAX_PER_RUN` | `5000` | Cap PMIDs per `/run` invocation |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | empty | If set, traces emitted via OTLP HTTP |
| `PORT` | `8080` | Cloud Run convention |

## Tests

```bash
cd ingest && go test ./internal/ingesters/pubmed/...
```

Network calls in tests are recorded with [go-vcr](https://github.com/dnaeon/go-vcr) under `testdata/cassettes/` (TODO: add cassettes).

## TODO

- Bulk baseline FTP fetch from `ftp.ncbi.nlm.nih.gov/pubmed/baseline/` for first-run seed (currently uses 7-day lookback).
- go-vcr cassette for esearch + efetch round-trips.
