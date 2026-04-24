# OpenAlex

| Field | Value |
|---|---|
| Tier | Citations |
| Records | ~250M works + ~4B citation edges |
| Access | Bulk S3 snapshot (`s3://openalex/data/`) + REST `api.openalex.org` |
| License | CC0 |
| Refresh | Bulk weekly; per-doc on demand |
| Implementation | [ingest/cmd/ingester-openalex/](../../ingest/cmd/ingester-openalex/) |

## Risks

- Bulk snapshot is ~300GB compressed; first-time ingest may take days. Stream-process directly from S3 without disk staging.
- ~4B citation edges → Neo4j PageRank refresh on full graph is the gating dependency for spec §19.3 backend choice.

## Topics

Emits citation edges to Pub/Sub `citation-edges` (separate topic from `raw-docs`); the indexer's Neo4j batcher consumes that topic directly.
