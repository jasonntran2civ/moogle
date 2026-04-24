# openFDA (drug + device)

| Field | Value |
|---|---|
| Tier | Regulatory |
| Records | ~15M+ device events; ~all FDA drug approvals; recall events |
| Access | REST `api.fda.gov/{drug,device}/...` |
| License | Public domain |
| Refresh | Drug approvals daily; recalls every 30min for priority lane; device events daily |
| Watermark | report_date / decision_date / submission_date |
| Implementation | [ingest/cmd/ingester-fda/](../../ingest/cmd/ingester-fda/) |

## Recall priority lane

`drug/enforcement` results also publish a `RecallEvent` to NATS `recall-fanout` for the gateway WebSocket subscribers. SLO: end-to-end ≤ 1 minute p95 (spec §14.1).
