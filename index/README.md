# indexer

Per spec §5.4. Consumes NATS `indexable-docs.>` and fans out to three batchers.

| Batcher | Trigger | Path |
|---|---|---|
| Meilisearch | 1000 docs OR 5s | [pkg/batchers/meili/](pkg/batchers/meili/) |
| Qdrant | 100 vectors OR 5s | [pkg/batchers/qdrant/](pkg/batchers/qdrant/) |
| Neo4j | 500 MERGE per tx OR 5s | [pkg/batchers/neo4jb/](pkg/batchers/neo4jb/) |

DLQ: after consumer's `MaxDeliver=5` failures, NATS publishes to `dlq.indexer`. See [docs/runbooks/indexer-dlq.md](../docs/runbooks/indexer-dlq.md).

## Idempotency

- Meilisearch: keyed by `id` (addOrReplace = upsert).
- Qdrant: keyed by `id` (PointsClient.Upsert).
- Neo4j: `MERGE (d:Document {id: $id})` — first-write-wins for properties (use SET to update).

## SLO

Spec §14.1: a record going `ingester → processor → indexer → searchable in all three indexes` within **5 minutes p95** over 24h. Alert at `IndexerLagAbove5Min` (see [infra/grafana/alerts/slo.yaml](../infra/grafana/alerts/slo.yaml)).

## Notes

- Qdrant batcher is currently a no-op write (logs only) until the official `github.com/qdrant/go-client` client is wired up. Marked TODO in source.
- Neo4j Cypher uses ORCID as author key when present, else display_name. Will need refinement when ORCID coverage drops.
