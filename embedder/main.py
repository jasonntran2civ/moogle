"""
EvidenceLens embedder — vLLM-backed BGE-M3 gRPC server (spec §5.3).

Bidirectional streaming gRPC. Dynamic batching (batch=32, max-wait=25ms)
for GPU throughput. CPU fallback to BGE-small-en-v1.5 (384-d) when GPU
unavailable.

Health probe via /healthz (FastAPI sidecar on a separate port for
Kubernetes-style liveness checks; gRPC Healthz also exposed via the
standard grpc.health.v1 service).
"""
from __future__ import annotations

import asyncio
import os
import signal
import time
from concurrent import futures
from contextlib import suppress
from dataclasses import dataclass

import grpc
import structlog
import uvicorn
from fastapi import FastAPI
from grpc_health.v1 import health, health_pb2_grpc
from grpc_reflection.v1alpha import reflection
from sentence_transformers import SentenceTransformer

# Generated stubs from proto/gen/python — committed at proto generate time.
# from evidencelens.v1 import embedder_pb2, embedder_pb2_grpc

log = structlog.get_logger("embedder")

# ---- Config ----

MODEL_NAME = os.getenv("EMBEDDING_MODEL", "BAAI/bge-m3")
FALLBACK_MODEL = os.getenv("EMBEDDING_FALLBACK_MODEL", "BAAI/bge-small-en-v1.5")
GRPC_PORT = int(os.getenv("GRPC_PORT", "50051"))
HEALTH_PORT = int(os.getenv("HEALTH_PORT", "8080"))
BATCH_SIZE = int(os.getenv("BATCH_SIZE", "32"))
MAX_WAIT_MS = int(os.getenv("MAX_WAIT_MS", "25"))


@dataclass
class ModelState:
    """Captures whether we're on the primary GPU model or CPU fallback."""
    name: str
    dim: int
    degraded: bool
    detail: str


def load_model() -> tuple[SentenceTransformer, ModelState]:
    """Try GPU + BGE-M3 first; fall back to CPU + BGE-small."""
    try:
        import torch
        if torch.cuda.is_available():
            log.info("loading primary model on GPU", model=MODEL_NAME)
            m = SentenceTransformer(MODEL_NAME, device="cuda")
            return m, ModelState(MODEL_NAME, m.get_sentence_embedding_dimension(), False, "")
    except Exception as e:  # noqa: BLE001
        log.warning("GPU init failed; falling back", error=str(e))

    log.warning("running CPU fallback", model=FALLBACK_MODEL)
    m = SentenceTransformer(FALLBACK_MODEL, device="cpu")
    return m, ModelState(FALLBACK_MODEL, m.get_sentence_embedding_dimension(),
                          True, "GPU unavailable; using CPU fallback")


# ---- Dynamic batching queue ----

@dataclass
class _Pending:
    request_id: str
    texts: list[str]
    future: asyncio.Future


class BatchedEmbedder:
    """Coalesces concurrent EmbedRequest calls into batched GPU inferences."""

    def __init__(self, model: SentenceTransformer, state: ModelState) -> None:
        self.model = model
        self.state = state
        self.queue: asyncio.Queue[_Pending] = asyncio.Queue()
        self._task: asyncio.Task | None = None

    async def start(self) -> None:
        self._task = asyncio.create_task(self._loop())

    async def stop(self) -> None:
        if self._task:
            self._task.cancel()
            with suppress(asyncio.CancelledError):
                await self._task

    async def embed(self, request_id: str, texts: list[str]) -> list[list[float]]:
        loop = asyncio.get_running_loop()
        fut: asyncio.Future = loop.create_future()
        await self.queue.put(_Pending(request_id, texts, fut))
        return await fut

    async def _loop(self) -> None:
        while True:
            batch: list[_Pending] = []
            try:
                first = await self.queue.get()
            except asyncio.CancelledError:
                return
            batch.append(first)
            deadline = time.monotonic() + MAX_WAIT_MS / 1000.0
            while sum(len(p.texts) for p in batch) < BATCH_SIZE:
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    break
                try:
                    p = await asyncio.wait_for(self.queue.get(), timeout=remaining)
                    batch.append(p)
                except asyncio.TimeoutError:
                    break

            flat = [t for p in batch for t in p.texts]
            try:
                vectors = await asyncio.to_thread(
                    self.model.encode,
                    flat,
                    batch_size=BATCH_SIZE,
                    normalize_embeddings=True,
                    convert_to_numpy=True,
                )
            except Exception as e:  # noqa: BLE001
                for p in batch:
                    p.future.set_exception(e)
                continue

            cursor = 0
            for p in batch:
                count = len(p.texts)
                p.future.set_result([v.tolist() for v in vectors[cursor:cursor + count]])
                cursor += count


# ---- gRPC servicer wired against the proto/gen/python shim ----

import sys as _sys, os as _os
_sys.path.insert(0, _os.path.join(_os.path.dirname(__file__), "..", "proto", "gen", "python"))

from evidencelens.v1 import (  # type: ignore[import]
    EmbedRequest, EmbedResponse, EmbeddingVector,
    EmbedderHealthzRequest, EmbedderHealthzResponse,
)
from evidencelens.v1.embedder_grpc import (  # type: ignore[import]
    EmbedderServiceServicer,
    add_EmbedderServiceServicer_to_server,
)


class EmbedderServicer(EmbedderServiceServicer):
    def __init__(self, batcher: "BatchedEmbedder") -> None:
        self.batcher = batcher

    async def Embed(self, request_iterator, context):  # type: ignore[override]
        async for req in request_iterator:
            vecs = await self.batcher.embed(req.request_id, list(req.texts))
            yield EmbedResponse(
                request_id=req.request_id,
                embeddings=[EmbeddingVector(values=v, dim=len(v)) for v in vecs],
                embedding_model=self.batcher.state.name,
            )

    async def EmbedOnce(self, request, context):  # type: ignore[override]
        vecs = await self.batcher.embed(request.request_id, list(request.texts))
        return EmbedResponse(
            request_id=request.request_id,
            embeddings=[EmbeddingVector(values=v, dim=len(v)) for v in vecs],
            embedding_model=self.batcher.state.name,
        )

    async def Healthz(self, request, context):  # type: ignore[override]
        return EmbedderHealthzResponse(
            status="degraded" if self.batcher.state.degraded else "ok",
            embedding_model=self.batcher.state.name,
            detail=self.batcher.state.detail,
        )


async def serve_grpc(model: SentenceTransformer, state: ModelState) -> grpc.aio.Server:
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=8))
    global _GLOBAL_BATCHER
    _GLOBAL_BATCHER = BatchedEmbedder(model, state)
    await _GLOBAL_BATCHER.start()
    add_EmbedderServiceServicer_to_server(EmbedderServicer(_GLOBAL_BATCHER), server)

    health_servicer = health.aio.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    await health_servicer.set("evidencelens.v1.EmbedderService",
                                health.HealthCheckResponse.SERVING if not state.degraded
                                else health.HealthCheckResponse.SERVING)

    server.add_insecure_port(f"[::]:{GRPC_PORT}")
    await server.start()
    log.info("grpc serving", port=GRPC_PORT, model=state.name, dim=state.dim, degraded=state.degraded)
    return server


# ---- HTTP shim (healthz + /embed for clients without proto stubs) ----

_GLOBAL_BATCHER: BatchedEmbedder | None = None


def make_app(state: ModelState) -> FastAPI:
    app = FastAPI()

    @app.get("/healthz")
    def healthz() -> dict:
        return {
            "status": "degraded" if state.degraded else "ok",
            "model": state.name,
            "dim": state.dim,
            "detail": state.detail,
        }

    @app.post("/embed")
    async def embed(body: dict) -> dict:
        request_id = str(body.get("request_id", "anon"))
        texts = list(body.get("texts", []))
        if _GLOBAL_BATCHER is None:
            return {"error": "batcher not ready"}
        vectors = await _GLOBAL_BATCHER.embed(request_id, texts)
        return {
            "request_id": request_id,
            "embeddings": [{"values": v, "dim": len(v)} for v in vectors],
            "embedding_model": state.name,
        }

    return app


# ---- Entry point ----

async def main() -> None:
    structlog.configure(
        processors=[
            structlog.processors.add_log_level,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.JSONRenderer(),
        ]
    )
    model, state = load_model()
    grpc_server = await serve_grpc(model, state)

    config = uvicorn.Config(make_app(state), host="0.0.0.0", port=HEALTH_PORT, log_level="info")
    http_server = uvicorn.Server(config)

    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, stop.set)

    try:
        await asyncio.gather(http_server.serve(), stop.wait())
    finally:
        log.info("shutting down")
        await grpc_server.stop(grace=10)


if __name__ == "__main__":
    asyncio.run(main())
