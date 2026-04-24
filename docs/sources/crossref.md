# CrossRef

| Field | Value |
|---|---|
| Tier | Literature enrichment |
| Records | ~140M DOI-indexed works |
| Access | REST `api.crossref.org/works/{doi}` |
| License | Public |
| Refresh | On-demand enrichment |
| Implementation | [ingest/cmd/ingester-crossref/](../../ingest/cmd/ingester-crossref/) |

## Notes

Polite pool requires `User-Agent: ... (mailto:...)`. Triggered by the processor when a record has a DOI but missing journal/publisher metadata.
