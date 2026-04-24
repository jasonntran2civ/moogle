# BigQuery — analytics dataset

Click events flow: frontend → click-logger Worker → pubsub-bridge → Pub/Sub `click-events` → BigQuery scheduled transfer → `analytics.clicks` (partitioned by `server_ts`, clustered by `variant` + `query_text`).

Free tier: 10GB storage + 1TB query/month. Aggregates are materialized nightly via [analytics_queries.sql](analytics_queries.sql) so the dashboard never ad-hoc scans the full partition.

## Schema

[../terraform/modules/gcp/clicks_schema.json](../terraform/modules/gcp/clicks_schema.json).

## Scheduled queries

Created by Terraform `google_bigquery_data_transfer_config` (TODO add to gcp module). Run nightly at 04:00 UTC.

## Cost monitoring

Set a billing alert at $0 (per [account-checklist.md](../../docs/account-checklist.md)). If query bytes/month approaches 1TB, prune retention or add more aggregates.
