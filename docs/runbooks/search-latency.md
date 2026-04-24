# Runbook: search p95 > 800ms

Triggered by `SearchLatencyP95Above800ms`.

## Quick checks

```bash
# Per-wave latency histogram in Grafana
# Dashboard: EvidenceLens — Overview > Search latency p95
```

## Likely culprits

| Symptom in trace | Cause | Action |
|---|---|---|
| BM25 sub-scorer slow | Meilisearch index unhealthy or query overly broad | Check MEILI_URL health; lower query topK |
| Vector sub-scorer slow | Qdrant HNSW thrashing or embedder cold-start | Restart embedder; check `degraded_components` |
| Citation sub-scorer slow | Neo4j PageRank query slow | Confirm refresh ran weekly; increase Neo4j heap |
| LTR rerank slow | Synthetic-label fallback path or model load | Confirm `LTR_MODEL_PATH` env points to a valid file |
| Gateway → scorer gRPC slow | Tunnel / network | Check Cloudflare Tunnel + Tailscale latencies |

## If sustained > 1h

Follow [search-availability.md](search-availability.md) escalation path; users start dropping searches at p95 > 1.5s.
