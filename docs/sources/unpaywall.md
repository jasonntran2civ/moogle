# Unpaywall

| Field | Value |
|---|---|
| Tier | Literature enrichment |
| Records | DOI → free OA PDF resolver |
| Access | REST `api.unpaywall.org/v2/{doi}?email=...` |
| License | CC0 |
| Refresh | On-demand |
| Implementation | [ingest/cmd/ingester-unpaywall/](../../ingest/cmd/ingester-unpaywall/) |

## Notes

Email parameter required. Triggered by the processor when a record has a DOI but no full-text URL.
