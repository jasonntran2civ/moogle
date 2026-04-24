# Runbook: indexer lag > 5min

Triggered by `IndexerLagAbove5Min` (gauge `indexer_lag_seconds > 300`).

## What it means

The time from a record landing in NATS `indexable-docs.*` to being searchable in all three indexes (Meili + Qdrant + Neo4j) exceeded 5 min p95. Spec §14.1 SLO breach.

## Triage

1. **Which batcher is slow?** The lag is dominated by the slowest of the three.
   ```bash
   docker logs evidencelens-indexer | grep '"flush"' | tail -50
   ```
2. **Backpressure?** Check `indexer_in_channel_dropped_total`. If non-zero, the indexer is rejecting submissions — usually means an upstream index is unhealthy.
3. **Batch sizes**: are the per-batcher size triggers being hit, or is everything time-triggered? If most flushes are time-triggered with low N, throughput is the issue (scale processor).

## Common causes

| Indicator | Cause | Fix |
|---|---|---|
| Meili flush count high, low N | Throughput bound on processor | Scale processor pipelines (`MAX_CONCURRENT_PIPELINES`) |
| Meili flush count low, slow flush | Meilisearch CPU-bound | Restart container; check disk |
| Qdrant flush slow | HNSW indexing thrashing | Pause writes; let `optimizers_config.indexing_threshold` settle |
| Neo4j flush slow | Lock contention on Author MERGE | Increase batcher size to amortize |

## Recovery

```bash
# Force flush everything
docker exec -it evidencelens-indexer kill -SIGUSR1 1   # if SIGUSR1 flush handler implemented
# Or restart cleanly
docker compose -f index/docker-compose.yml restart indexer
```
