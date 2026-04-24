# Runbook: indexer dead-letter spike

Triggered by alert `IndexerDLQGrowing` (rate > 0.1 msg/sec for 15 min).

## Symptom

Records appearing on NATS subject `dlq.indexer`. The first line of each DLQ message is a `# <reason>` comment with the failure cause.

## Triage

```bash
# Inspect last 50 dead-lettered messages
docker exec -it el-nats nats stream view dlq.indexer --raw | head -100

# Count by reason prefix
docker exec -it el-nats nats stream view dlq.indexer --raw \
  | grep '^# ' | sort | uniq -c | sort -rn
```

Common causes:

| Reason prefix | Likely cause | Action |
|---|---|---|
| `unmarshal: ...` | Processor wrote a malformed envelope | Check processor logs; fix parser; replay DLQ |
| `meili add docs: ...` | Meilisearch unhealthy or schema mismatch | `docker logs el-meilisearch`; restart if needed |
| `neo4j upsert: ...` | Neo4j down or transaction conflict | Check `el-neo4j` health; retry |
| `qdrant upsert: ...` | Qdrant down or wrong collection dim | Verify collection `evidence_v1` is 1024-d (BGE-M3) |

## Replay

```bash
# Once root cause is fixed, drain DLQ back into indexable-docs.{source}
docker exec -it el-nats nats stream view dlq.indexer --raw | while read line; do
  if [[ ! "$line" =~ ^# ]]; then
    # Parse source from JSON, republish
    src=$(echo "$line" | jq -r '.document.source')
    docker exec -i el-nats nats pub "indexable-docs.$src" "$line"
  fi
done

# Then purge the DLQ
docker exec -it el-nats nats stream purge dlq.indexer
```

## Prevention

- Pre-merge: any change to indexer batcher logic requires the integration test in `tests/integration/indexer_dlq_test.go` to pass.
- Pre-deploy: smoke test against dev compose with a known-good 100-doc fixture.
