# PubMed (source)

| Field | Value |
|---|---|
| Tier | Literature |
| Records | ~38M citations + abstracts |
| Access | NCBI E-utilities REST API + FTP bulk baseline |
| License | Public domain (NLM) |
| Refresh | Baseline once, daily updates |
| Rate limit | 3/sec without key, 10/sec with API key |
| Watermark | `EDAT` (entry date), ISO-8601 |
| Implementation | [ingest/cmd/ingester-pubmed/](../../ingest/cmd/ingester-pubmed/) |

## Notes

- Get a free API key at <https://www.ncbi.nlm.nih.gov/account/settings/> to raise the rate limit.
- NCBI ToS requires `tool=` and `email=` query parameters on all requests.
- The baseline FTP files at `ftp.ncbi.nlm.nih.gov/pubmed/baseline/` are ~38M records compressed; first-run bootstrap should stream-process them rather than fetch via E-utilities (which would take weeks).
- EDAT (entry date) is more useful than PUBDATE (publication date) for delta ingestion because PubMed records can be updated long after publication.
