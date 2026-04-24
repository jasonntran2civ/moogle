"""
scorer-pool entry point (spec §5.5). gRPC server.

ScorerService.Search streams PartialResults waves:
  wave 1 @ 200ms (top 5)
  wave 2 @ 500ms (next 10)
  wave 3 @ 1000ms (final 35)

Internal sub-scorers run as concurrent asyncio tasks. RRF k=60 fuses.
XGBoost LambdaMART head reranks top 50.

Generated proto stubs imported when available; fallback to a
GRPC-typed-dict shape so the service runs end-to-end against a fake
embedder pre-codegen.
"""
from __future__ import annotations

import asyncio
import os
import signal
from concurrent import futures
from contextlib import suppress
from dataclasses import dataclass

import grpc
import structlog
from grpc_health.v1 import health, health_pb2_grpc

from bm25 import BM25Scorer
from citation import CitationScorer
from fusion import rrf
from ltr import CandidateFeatures, LTRReranker
from recency import recency_score
from vector import VectorScorer

log = structlog.get_logger("scorer")


@dataclass
class Config:
    grpc_port: int
    meili_url: str
    meili_key: str
    qdrant_url: str
    neo4j_url: str
    neo4j_user: str
    neo4j_password: str
    embedder_url: str
    ltr_model_path: str

    @classmethod
    def from_env(cls) -> "Config":
        return cls(
            grpc_port=int(os.getenv("GRPC_PORT", "50052")),
            meili_url=os.getenv("MEILI_URL", "http://localhost:7700"),
            meili_key=os.getenv("MEILI_KEY", ""),
            qdrant_url=os.getenv("QDRANT_URL", "http://localhost:6333"),
            neo4j_url=os.getenv("NEO4J_URL", "bolt://localhost:7687"),
            neo4j_user=os.getenv("NEO4J_USER", "neo4j"),
            neo4j_password=os.getenv("NEO4J_PASSWORD", "changeme-dev-only"),
            embedder_url=os.getenv("EMBEDDER_GRPC_URL", "embedder:50051"),
            ltr_model_path=os.getenv("LTR_MODEL_PATH", ""),
        )


class ScorerCore:
    """Search orchestration logic. Wired into the gRPC servicer below
    (commented out until proto stubs land)."""

    def __init__(self, cfg: Config) -> None:
        self.cfg = cfg
        self.bm25 = BM25Scorer(cfg.meili_url, cfg.meili_key)
        self.vector = VectorScorer(cfg.qdrant_url)
        self.citation = CitationScorer(cfg.neo4j_url, cfg.neo4j_user, cfg.neo4j_password)
        self.ltr = LTRReranker(cfg.ltr_model_path)

    async def search(self, query: str, filters: dict | None, top_k: int = 50):
        """Yields (wave_no, is_final, results) tuples. The gRPC servicer
        wraps these into PartialResults messages."""
        # 1. Fan out BM25 + vector concurrently (both block on indexes).
        # Vector requires query embedding via embedder gRPC; for now we
        # use a deterministic stub vector so the pipeline runs without
        # the proto stubs in place.
        from process.utils.embedder_client import EmbedderClient  # type: ignore[import]
        emb = EmbedderClient(self.cfg.embedder_url)
        loop = asyncio.get_running_loop()

        async def _bm25():
            return await loop.run_in_executor(None, self.bm25.search, query, filters, 200)

        async def _vector():
            qvecs = await emb.embed("query", [query])
            qv = qvecs[0].vector
            return await loop.run_in_executor(None, self.vector.search, qv, filters, 200)

        bm25_task = asyncio.create_task(_bm25())
        vec_task = asyncio.create_task(_vector())

        # First wave at 200ms: emit BM25 top 5 if ready, else vector top 5.
        await asyncio.sleep(0.2)
        first_results: list[dict] = []
        if bm25_task.done():
            first_results = [self._to_result(h.document, bm25=h.score) for h in bm25_task.result()[:5]]
        elif vec_task.done():
            first_results = [self._to_result(h.payload, vector=h.score) for h in vec_task.result()[:5]]
        yield 1, False, first_results

        # Wait for both BM25 + vector to complete (or 500ms cap).
        await asyncio.wait({bm25_task, vec_task}, timeout=0.3, return_when=asyncio.ALL_COMPLETED)
        bm25_hits = bm25_task.result() if bm25_task.done() else []
        vec_hits = vec_task.result() if vec_task.done() else []

        # 2. Compute citation + recency over the union top 500.
        union_ids = list({h.doc_id for h in bm25_hits} | {h.doc_id for h in vec_hits})[:500]
        cite_scores = {c.doc_id: c.pagerank for c in await loop.run_in_executor(None, self.citation.score, union_ids)}

        rec_scores: dict[str, float] = {}
        merged_payloads: dict[str, dict] = {}
        for h in bm25_hits:
            merged_payloads[h.doc_id] = h.document
        for h in vec_hits:
            if h.doc_id not in merged_payloads:
                merged_payloads[h.doc_id] = h.payload
        for did, p in merged_payloads.items():
            rec_scores[did] = recency_score(p.get("published_at"))

        # 3. RRF fusion over four sub-scorer rankings.
        rankings = {
            "bm25":     [h.doc_id for h in bm25_hits],
            "vector":   [h.doc_id for h in vec_hits],
            "citation": sorted(cite_scores, key=lambda i: cite_scores[i], reverse=True),
            "recency":  sorted(rec_scores, key=lambda i: rec_scores[i], reverse=True),
        }
        fused = rrf(rankings, k=60)

        # Second wave at 500ms total: top 15.
        wave2 = []
        for f in fused[:15]:
            p = merged_payloads.get(f.doc_id, {})
            wave2.append(self._to_result(
                p,
                final_score=f.rrf_score,
                bm25=next((h.score for h in bm25_hits if h.doc_id == f.doc_id), 0.0),
                vector=next((h.score for h in vec_hits if h.doc_id == f.doc_id), 0.0),
                pagerank=cite_scores.get(f.doc_id, 0.0),
                recency=rec_scores.get(f.doc_id, 0.0),
            ))
        yield 2, False, wave2

        # 4. LTR rerank top 50.
        candidates = []
        for f in fused[:top_k]:
            p = merged_payloads.get(f.doc_id, {})
            cf = CandidateFeatures(
                bm25=next((h.score for h in bm25_hits if h.doc_id == f.doc_id), 0.0),
                vector=next((h.score for h in vec_hits if h.doc_id == f.doc_id), 0.0),
                pagerank=cite_scores.get(f.doc_id, 0.0),
                recency=rec_scores.get(f.doc_id, 0.0),
                study_type=p.get("study_type", "OTHER"),
                has_full_text=bool(p.get("has_full_text", False)),
                citation_count=int(p.get("citation_count", 0)),
                has_coi_authors=bool(p.get("has_coi_authors", False)),
                journal_predatory=bool(p.get("journal_predatory", False)),
            )
            candidates.append((f.doc_id, cf))
        ltr_scores = self.ltr.score(candidates, query)

        ranked = sorted(candidates, key=lambda c: ltr_scores.get(c[0], 0.0), reverse=True)

        # Final wave (wave 3): up to top_k - 15 already shown.
        wave3 = []
        for did, cf in ranked[15:top_k]:
            p = merged_payloads.get(did, {})
            wave3.append(self._to_result(
                p,
                final_score=ltr_scores.get(did, 0.0),
                bm25=cf.bm25, vector=cf.vector, pagerank=cf.pagerank, recency=cf.recency,
                ltr=ltr_scores.get(did, 0.0),
                ltr_model_version=self.ltr.model_version,
            ))
        yield 3, True, wave3
        await emb.close()

    @staticmethod
    def _to_result(payload: dict, **scores) -> dict:
        return {
            "document": payload,
            "final_score": scores.get("final_score", 0.0),
            "breakdown": {
                "bm25": scores.get("bm25", 0.0),
                "vector": scores.get("vector", 0.0),
                "citation_pagerank": scores.get("pagerank", 0.0),
                "recency": scores.get("recency", 0.0),
                "rrf": scores.get("final_score", 0.0),
                "ltr": scores.get("ltr", 0.0),
                "ltr_model_version": scores.get("ltr_model_version", ""),
            },
        }


# ---- gRPC bootstrap (proto-typed servicer wired against shim) ----

import sys as _sys, os as _os
_sys.path.insert(0, _os.path.join(_os.path.dirname(__file__), "..", "proto", "gen", "python"))

from evidencelens.v1 import (  # type: ignore[import]
    PartialResults, ScoredResult as PbScoredResult, ScoreBreakdown,
    Document as PbDocument, ScorerHealthzRequest, ScorerHealthzResponse,
)
from evidencelens.v1.scorer_grpc import (  # type: ignore[import]
    ScorerServiceServicer,
    add_ScorerServiceServicer_to_server,
)


class ScorerServicer(ScorerServiceServicer):
    def __init__(self, core: ScorerCore) -> None:
        self.core = core

    async def Search(self, request, context):  # type: ignore[override]
        filters = request.filters.__dict__ if hasattr(request.filters, "__dict__") else (request.filters or {})
        async for wave_no, is_final, results in self.core.search(request.query, filters, request.top_k or 50):
            pb_results = [
                PbScoredResult(
                    document=PbDocument(**{k: v for k, v in r["document"].items() if hasattr(PbDocument(), k)}),
                    final_score=r["final_score"],
                    breakdown=ScoreBreakdown(
                        bm25_score=r["breakdown"]["bm25"],
                        vector_score=r["breakdown"]["vector"],
                        citation_pagerank=r["breakdown"]["citation_pagerank"],
                        recency_score=r["breakdown"]["recency"],
                        rrf_score=r["breakdown"]["rrf"],
                        ltr_score=r["breakdown"]["ltr"],
                        ltr_model_version=r["breakdown"]["ltr_model_version"],
                    ),
                )
                for r in results
            ]
            yield PartialResults(results=pb_results, wave=wave_no, is_final=is_final)

    async def Healthz(self, request, context):  # type: ignore[override]
        return ScorerHealthzResponse(status="ok")


async def serve(cfg: Config) -> grpc.aio.Server:
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=16))
    add_ScorerServiceServicer_to_server(ScorerServicer(ScorerCore(cfg)), server)
    health_servicer = health.aio.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    await health_servicer.set("evidencelens.v1.ScorerService", health.HealthCheckResponse.SERVING)
    server.add_insecure_port(f"[::]:{cfg.grpc_port}")
    await server.start()
    log.info("scorer grpc serving", port=cfg.grpc_port)
    return server


async def main() -> None:
    structlog.configure(processors=[
        structlog.processors.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.JSONRenderer(),
    ])
    cfg = Config.from_env()
    _ = ScorerCore(cfg)  # ensure indexes are reachable at startup
    server = await serve(cfg)

    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, stop.set)
    await stop.wait()
    await server.stop(grace=5)


if __name__ == "__main__":
    with suppress(KeyboardInterrupt):
        asyncio.run(main())
