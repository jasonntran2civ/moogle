# NIH RePORTER

| Field | Value |
|---|---|
| Tier | Funding |
| Records | All federally-funded biomedical grants |
| Access | REST `api.reporter.nih.gov/v2/projects/search` |
| License | Public |
| Refresh | Weekly |
| Implementation | [ingest/cmd/ingester-nih-reporter/](../../ingest/cmd/ingester-nih-reporter/) |

## Notes

Joined to documents via `funding.grant_id` lookups in the processor.
