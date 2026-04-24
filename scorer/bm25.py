"""BM25 sub-scorer — Meilisearch (spec §5.5)."""
from __future__ import annotations

from dataclasses import dataclass
from typing import Any

import meilisearch


@dataclass
class BM25Hit:
    doc_id: str
    score: float
    document: dict[str, Any]


class BM25Scorer:
    def __init__(self, url: str, api_key: str, index_name: str = "documents") -> None:
        self.client = meilisearch.Client(url, api_key)
        self.index = self.client.index(index_name)

    def search(self, query: str, filters: dict | None = None, top_k: int = 200) -> list[BM25Hit]:
        params: dict[str, Any] = {
            "limit": top_k,
            "showRankingScore": True,
            "attributesToRetrieve": ["id", "title", "abstract", "study_type", "published_year",
                                       "citation_count", "citation_pagerank", "has_coi_authors",
                                       "license", "source", "authors_display"],
        }
        if filters:
            params["filter"] = self._build_filter(filters)
        res = self.index.search(query, params)
        hits: list[BM25Hit] = []
        for h in res.get("hits", []):
            hits.append(BM25Hit(
                doc_id=h["id"],
                score=float(h.get("_rankingScore", 0.0)),
                document=h,
            ))
        return hits

    @staticmethod
    def _build_filter(f: dict) -> list[str]:
        out: list[str] = []
        if f.get("study_types"):
            out.append("study_type IN " + str(list(f["study_types"])))
        if f.get("published_year_min"):
            out.append(f"published_year >= {int(f['published_year_min'])}")
        if f.get("published_year_max"):
            out.append(f"published_year <= {int(f['published_year_max'])}")
        if f.get("only_with_coi"):
            out.append("has_coi_authors = true")
        if f.get("only_with_full_text"):
            out.append("has_full_text = true")
        if f.get("exclude_predatory_journals"):
            out.append("journal_predatory = false")
        return out
