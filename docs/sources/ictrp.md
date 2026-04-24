# WHO ICTRP

| Field | Value |
|---|---|
| Tier | Trials |
| Records | ~800k global trials (EU CTIS, ChiCTR, CTRI, JPRN, ANZCTR, etc.) |
| Access | Bulk weekly XML at trialsearch.who.int |
| License | Per source |
| Refresh | Weekly |
| Watermark | snapshot hash |
| Implementation | [ingest/cmd/ingester-ictrp/](../../ingest/cmd/ingester-ictrp/) |

## Notes

Diff against previous weekly snapshot (stored in R2) to emit only new/changed trials.
