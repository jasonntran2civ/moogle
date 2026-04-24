# Cochrane Library

| Field | Value |
|---|---|
| Tier | Literature |
| Records | ~9k systematic reviews |
| Access | RSS feed + DOI resolution via CrossRef for abstracts |
| License | Per review (most academic-only — never serve full text) |
| Refresh | Monthly |
| Implementation | [ingest/cmd/ingester-cochrane/](../../ingest/cmd/ingester-cochrane/) |

## Compliance

EvidenceLens stores Cochrane metadata + abstracts only, never full text. Result cards link out to the Cochrane Library record.
