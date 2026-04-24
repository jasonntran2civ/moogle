# USPSTF + NICE + AHRQ guidelines

| Field | Value |
|---|---|
| Tier | Guidelines |
| Records | ~hundreds across three sources |
| Access | HTML scraping via Colly |
| License | Public (USPSTF, AHRQ) / OGL UK (NICE) |
| Refresh | Monthly |
| Implementation | [ingest/cmd/ingester-guidelines/](../../ingest/cmd/ingester-guidelines/) |

## Notes

Per-source crawl rules in YAML config. Render-to-markdown via gomarkdownify before R2 archive.
