# Runbook: recall fanout SLO breach

Triggered by alert `RecallFanoutAbove1Min` (p95 > 60s for 5 min).

The flagship surveillance feature: an FDA recall must reach all WS subscribers in < 1min p95 (spec §14.1).

## Where to look first

1. **fda-ingester recall lane** — is the priority publish path emitting `RecallEvent` to NATS recall-fanout?
   ```bash
   docker exec -it el-nats nats stream view EVIDENCELENS --filter 'recall-fanout' --raw | tail -20
   ```
2. **gateway WebSocket subscription bridge** — is the bridge from NATS to WS subscribers running?
   ```bash
   docker logs evidencelens-gateway | grep -i recall
   ```
3. **Cloudflare Tunnel** — is the gateway reachable from the public WSS endpoint?

## Common causes

- **fda-ingester scheduler interval too long**: cron is `*/30 * * * *` (every 30min) per Terraform, which sets a hard floor on detection latency. If the spec demands faster, change in `infra/terraform/modules/gcp/main.tf`.
- **NATS subject not bridged to WS**: `gateway/src/ws/ws.module.ts subscribe` handler stub. Real impl is TODO.
- **WS subscribers all disconnected**: not actually a bug; nothing to fan out to. Check `evidencelens_ws_active_connections` in Grafana.

## Manual replay

```bash
# Inject a synthetic recall to test the path end-to-end
docker exec -i el-nats nats pub recall-fanout '{"recall_id":"test-001","agency":"fda","product_name":"Test","drug_class":"SGLT2","recall_class":"II","emitted_at":"2026-04-24T12:00:00Z"}'
```
