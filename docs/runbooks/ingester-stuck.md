# Runbook: ingester stuck

Triggered by alert `IngesterStuck` (last_run_at older than expected period for source).

## Triage

```bash
# Check ingestion_state
psql $DATABASE_URL -c \
  "SELECT source, last_run_at, status, last_error FROM ingestion_state ORDER BY last_run_at NULLS FIRST"

# Check Cloud Run service status
gcloud run services describe ingester-{source} --region us-central1

# Check recent invocations
gcloud logging read 'resource.type="cloud_run_revision" AND resource.labels.service_name="ingester-{source}"' --limit 50
```

## Common causes

1. **Upstream API down** — GitHub status page for NCBI / openFDA / etc. Wait + retry; if persistent, file an issue.
2. **Rate limit hit** — check `last_error` for 429. Halve concurrency, wait an hour, resume.
3. **Cloud Run scheduler missed invocation** — `gcloud scheduler jobs describe ingester-{source}-cron`. Trigger manually: `gcloud scheduler jobs run ingester-{source}-cron`.
4. **Postgres watermark write failed** — check Postgres logs. Usually disk full on TrueNAS.

## Recovery

```bash
# Manual trigger
curl -X POST https://ingester-{source}-...run.app/run -H "Authorization: Bearer $(gcloud auth print-identity-token)"

# Reset watermark to N days ago
psql $DATABASE_URL -c "UPDATE ingestion_state SET last_high_watermark = '2026-04-01', status = 'idle' WHERE source = '{source}'"
```
