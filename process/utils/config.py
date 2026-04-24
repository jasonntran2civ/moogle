"""Centralized env-var config for the processor (mirrors Moogle pattern).

Pure-functional: no global state, every service reads via this module so
mocking in tests is trivial.
"""
from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    pg_dsn: str
    nats_url: str
    redis_url: str
    pubsub_project: str
    pubsub_sub_raw_docs: str
    r2_endpoint: str
    r2_access_key: str
    r2_secret_key: str
    r2_bucket: str
    embedder_grpc_url: str
    open_payments_lookup_url: str
    max_concurrent_pipelines: int
    chunk_tokens: int
    chunk_overlap: int
    fuzzy_min_confidence: float
    cache_ttl_days: int

    @classmethod
    def from_env(cls) -> "Config":
        return cls(
            pg_dsn=_must("DATABASE_URL"),
            nats_url=os.getenv("NATS_URL", "nats://localhost:4222"),
            redis_url=os.getenv("REDIS_URL", "redis://localhost:6379/0"),
            pubsub_project=_must("GCP_PROJECT"),
            pubsub_sub_raw_docs=os.getenv("PUBSUB_SUBSCRIPTION_RAW_DOCS", "raw-docs.processor"),
            r2_endpoint=_must("R2_ENDPOINT"),
            r2_access_key=_must("R2_ACCESS_KEY_ID"),
            r2_secret_key=_must("R2_SECRET_ACCESS_KEY"),
            r2_bucket=_must("R2_BUCKET"),
            embedder_grpc_url=os.getenv("EMBEDDER_GRPC_URL", "embedder:50051"),
            open_payments_lookup_url=os.getenv(
                "OPEN_PAYMENTS_LOOKUP_URL",
                "http://ingester-open-payments:8080/lookup",
            ),
            max_concurrent_pipelines=int(os.getenv("MAX_CONCURRENT_PIPELINES", "50")),
            chunk_tokens=int(os.getenv("CHUNK_TOKENS", "512")),
            chunk_overlap=int(os.getenv("CHUNK_OVERLAP", "64")),
            fuzzy_min_confidence=float(os.getenv("FUZZY_MIN_CONFIDENCE", "0.90")),
            cache_ttl_days=int(os.getenv("AUTHOR_PAYMENT_CACHE_TTL_DAYS", "30")),
        )


def _must(name: str) -> str:
    v = os.environ.get(name, "")
    if not v:
        raise RuntimeError(f"required env var not set: {name}")
    return v
