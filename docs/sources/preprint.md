# bioRxiv + medRxiv

| Field | Value |
|---|---|
| Tier | Literature |
| Records | ~150k bioRxiv + ~50k medRxiv |
| Access | REST API at api.biorxiv.org and api.medrxiv.org (same shape) |
| License | CC-BY (bioRxiv) / CC-BY-NC-ND (medRxiv default) |
| Refresh | Daily |
| Watermark | ISO date |
| Implementation | [ingest/cmd/ingester-preprint/](../../ingest/cmd/ingester-preprint/) |

## Notes

One ingester serves both via Servers config. Cursor-paginated. Free, no key required.
