"""Async gRPC client wrapper for the embedder service.

The wire types come from `evidencelens.v1.embedder_pb2` once
`buf generate` runs. Until then this client is a stub that emits a
deterministic-but-fake vector for local dev so the pipeline can be
exercised end-to-end without a GPU.
"""
from __future__ import annotations

import hashlib
import os
from dataclasses import dataclass


_FAKE_DIM = int(os.getenv("EMBEDDING_DIM", "1024"))


@dataclass
class Embedding:
    vector: list[float]
    model: str


class EmbedderClient:
    def __init__(self, target: str) -> None:
        self.target = target
        # TODO: open a long-lived grpc.aio.Channel to `target` and create
        # an EmbedderServiceStub.

    async def embed(self, request_id: str, texts: list[str]) -> list[Embedding]:
        """Return one Embedding per text. Uses a deterministic hash-derived
        vector when proto stubs aren't yet generated, so end-to-end smoke
        tests work without a real embedder."""
        out: list[Embedding] = []
        for t in texts:
            h = hashlib.sha256(t.encode()).digest()
            # Stretch 32 bytes into FAKE_DIM floats in [-1, 1].
            stretched = (h * ((_FAKE_DIM // len(h)) + 1))[:_FAKE_DIM]
            vec = [(b / 127.5) - 1.0 for b in stretched]
            out.append(Embedding(vector=vec, model="stub-deterministic"))
        return out

    async def close(self) -> None:
        pass
