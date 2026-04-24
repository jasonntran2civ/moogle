# Event Topic Catalog

EvidenceLens uses two event carriers: **GCP Pub/Sub** (cross-cloud, raw
ingest fan-in and click telemetry) and **NATS JetStream** (in-cluster on
the NAS, indexable docs + recall fanout + DLQ). This document is the
contract for both.

**Spec source of truth:** [`docs/EVIDENCELENS_SPEC.md`](../EVIDENCELENS_SPEC.md) §3.6, §4.

## Routing diagram

```
ingesters (Cloud Run)
   ├─ raw bytes ─────────────────────────► R2 raw/{source}/{date}/{id}.json.gz
   └─ RawDocEvent ───► Pub/Sub raw-docs ──► pubsub-bridge Worker ──► NATS raw-docs.{source}
                                                                          │
                                                                          ▼
                                                                  processor (NAS)
                                                                          │
                       open-payments-ingester /lookup ◄──────────── author-payment-joiner
                       embedder gRPC          ◄──────────────────── chunk → embed
                                                                          │
                                  IndexableDocEvent                       │
                                                                          ▼
                                       ┌────────────────────── NATS indexable-docs.{source}
                                       │
                                       ▼
                                indexer (NAS) ──► Meilisearch / Qdrant / Neo4j
                                       │
                                       └──── on 5 failures ────► NATS dlq.indexer

fda-ingester (recall path) ─── RecallEvent ───► NATS recall-fanout ──► gateway WS subscribers

frontend → click-logger Worker ─ ClickEvent ──► Pub/Sub click-events ──► BigQuery analytics.clicks
```

## GCP Pub/Sub topics

| Topic | Schema | Publisher | Subscribers | Notes |
|---|---|---|---|---|
| `raw-docs` | `RawDocEvent` | All 12 ingesters | `pubsub-bridge` Cloudflare Worker | At-least-once. Bridge ACKs after NATS publish. Free-tier 10GB/mo egress; messages reference R2 raw archive rather than embedding bytes. |
| `click-events` | `ClickEvent` | `click-logger` Cloudflare Worker | BigQuery scheduled transfer | Batched in the Worker (size 50 / time 5s). |
| `citation-edges` | `(citing_doc_id, cited_doc_id, edge_type)` JSON | `openalex-citations` ingester | NATS bridge → indexer (Neo4j batcher) | Wire format JSON, not protobuf — the schema is too narrow to justify proto codegen. |

**Subscriptions:**

- `raw-docs.bridge`: push subscription with OIDC JWT to `pubsub-bridge` Worker (`POST /pubsub/raw-docs`).
- `click-events.bigquery`: BigQuery scheduled transfer, 5-minute interval, target table `analytics.clicks`.
- `citation-edges.bridge`: push subscription with OIDC JWT to `pubsub-bridge` Worker.

## NATS JetStream subjects

Stream name: `EVIDENCELENS`. Single stream, retention `WorkQueue`,
max_age 7 days, replicas 1 (single NAS node).

| Subject | Schema | Publisher | Consumers | Notes |
|---|---|---|---|---|
| `raw-docs.{source}` | `RawDocEvent` | `pubsub-bridge` (forwarded from Pub/Sub) | `processor` (durable consumer `processor-{source}`) | One subject per source for parallel consumption. |
| `indexable-docs.{source}` | `IndexableDocEvent` | `processor` | `indexer` (durable consumer `indexer`) | Wildcard subscription `indexable-docs.>`. |
| `recall-fanout` | `RecallEvent` | `fda-ingester` (priority lane) | `gateway` (push to WS subscribers) | SLO ≤ 1min E2E (spec §14.1). At-least-once with dedupe key = `recall_id`. |
| `dlq.indexer` | `IndexableDocEvent` + error context | `indexer` after 5 retries | Operator-driven (manual replay tool) | See [docs/runbooks/indexer-dlq.md](../runbooks/indexer-dlq.md). |

### Consumer policies

```jsonc
// processor durable consumer
{
  "name": "processor-pubmed",
  "filter_subject": "raw-docs.pubmed",
  "ack_policy": "explicit",
  "ack_wait": "60s",
  "max_deliver": 5,
  "max_ack_pending": 100
}

// indexer durable consumer
{
  "name": "indexer",
  "filter_subject": "indexable-docs.>",
  "ack_policy": "explicit",
  "ack_wait": "30s",
  "max_deliver": 5,
  "max_ack_pending": 1000
}
```

## Message size limits

- Pub/Sub `raw-docs`: ≤ 10 KB per message (only metadata + R2 pointer; the
  actual raw response lives in R2).
- NATS `indexable-docs.*`: typical 50–200 KB (full Document including
  abstract, mesh terms, embedding 1024×4=4KB). Hard cap 1 MB; oversized
  messages dead-letter to `dlq.indexer` immediately.
- NATS `recall-fanout`: ≤ 2 KB.
- Pub/Sub `click-events`: ≤ 1 KB.

## Schemas

All Protobuf schemas in [`proto/evidencelens/v1/events.proto`](../../proto/evidencelens/v1/events.proto).
JSON-on-the-wire schemas (`citation-edges`) inline above. Generated
language-specific stubs in [`proto/gen/`](../../proto/gen/).

## Versioning

Topic and subject names are pinned to v1. Breaking changes ship under
`raw-docs.v2.{source}` etc., with a migration window during which
producers double-publish and consumers read both. Consult the RFC
process in [docs/rfcs/README.md](../rfcs/README.md) before any change.

## Backpressure & flow control

- Pub/Sub: built-in. Cloud Run ingester scales down when push subscription
  ack rate drops (via Pub/Sub flow control settings).
- NATS: `max_ack_pending` caps in-flight messages per consumer. The
  processor signals upstream by pausing its Pub/Sub pull when its NATS
  publish rate exceeds the indexer's drain rate (mirrors Moogle's spider
  backpressure pattern at [services/spider/cmd/spider/main.go](../../../moogle/services/spider/cmd/spider/main.go)).
