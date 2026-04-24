"""Vector sub-scorer — Qdrant + embedder gRPC (spec §5.5)."""
from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from qdrant_client import QdrantClient
from qdrant_client.http.models import Filter, FieldCondition, MatchValue, MatchAny, Range


@dataclass
class VectorHit:
    doc_id: str
    score: float
    payload: dict[str, Any]


class VectorScorer:
    def __init__(self, url: str, collection: str = "evidence_v1") -> None:
        self.client = QdrantClient(url=url)
        self.collection = collection

    def search(
        self,
        query_vector: list[float],
        filters: dict | None = None,
        top_k: int = 200,
    ) -> list[VectorHit]:
        flt = self._build_filter(filters or {})
        res = self.client.search(
            collection_name=self.collection,
            query_vector=query_vector,
            limit=top_k,
            query_filter=flt,
            with_payload=True,
        )
        return [VectorHit(doc_id=str(p.payload.get("doc_id", p.id)), score=float(p.score), payload=dict(p.payload or {})) for p in res]

    @staticmethod
    def _build_filter(f: dict) -> Filter | None:
        must: list[FieldCondition] = []
        if f.get("study_types"):
            must.append(FieldCondition(key="study_type", match=MatchAny(any=list(f["study_types"]))))
        if f.get("only_with_coi"):
            must.append(FieldCondition(key="has_coi_authors", match=MatchValue(value=True)))
        if f.get("published_year_min") or f.get("published_year_max"):
            must.append(FieldCondition(
                key="published_year",
                range=Range(gte=f.get("published_year_min"), lte=f.get("published_year_max")),
            ))
        if not must:
            return None
        return Filter(must=must)
