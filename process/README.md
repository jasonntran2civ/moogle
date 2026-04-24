# processor

Per spec §5.2. Consumes `raw-docs.>` from NATS (forwarded from Pub/Sub by `pubsub-bridge` Worker), runs the 7-stage pipeline, publishes `IndexableDocEvent` to `indexable-docs.{source}`.

## Pipeline

1. Parse — dispatch to per-source parser (`parsers/*.py`).
2. Normalize — Unicode NFC, MeSH ID canonicalization (TODO).
3. Entity link — scispaCy `en_core_sci_lg` + UMLS (TODO; stubbed).
4. Chunk — sliding window 512 tokens / 64 overlap (`utils/chunker.py`).
5. Embed — gRPC stream to `embedder` (`utils/embedder_client.py`; stubbed with deterministic fake vector until proto stubs land).
6. **Author × Open Payments fuzzy join** (`utils/author_payment_joiner.py`) — flagship logic. Conservative threshold ≥ 0.90; state-restricted when affiliation known. Cached 30 days in Postgres `author_payment_cache`.
7. Predatory journal flag (TODO).
8. Publish to NATS `indexable-docs.{source}`.

## Concurrency

`MAX_CONCURRENT_PIPELINES=50` controls the in-flight cap. Backpressure: NATS publish lag check pauses Pub/Sub pull (TODO).

## Run

```bash
uv sync
DATABASE_URL=... NATS_URL=... GCP_PROJECT=... R2_ENDPOINT=... \
R2_ACCESS_KEY_ID=... R2_SECRET_ACCESS_KEY=... R2_BUCKET=evidencelens-raw \
EMBEDDER_GRPC_URL=embedder:50051 \
OPEN_PAYMENTS_LOOKUP_URL=http://ingester-open-payments:8080/lookup \
uv run python main.py
```

## Notes

- Chunker uses tiktoken (cl100k_base) as a tokenization proxy — close to BGE-M3 within ±10% for budgeting. Real tokenizer used in `embedder` for the actual cut.
- Author × Open Payments lookup defends against high false-positive rate by skipping initials-only authors when state is unknown.
