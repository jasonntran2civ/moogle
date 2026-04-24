"""bioRxiv / medRxiv API JSON parser. Stub."""
from __future__ import annotations

import json
from typing import Any


def parse(raw: bytes) -> dict[str, Any]:
    r = json.loads(raw)
    server = r.get("server", "biorxiv")
    doi = r.get("doi", "")
    return {
        "id": f"{server}:{doi}",
        "source": server,
        "source_native_id": doi,
        "doi": doi.lower() if doi else None,
        "title": r.get("title", ""),
        "abstract": r.get("abstract", ""),
        "canonical_url": f"https://www.{server}.org/content/{doi}",
        "license": r.get("license") or ("CC-BY-4.0" if server == "biorxiv" else "CC-BY-NC-ND-4.0"),
        "study_type": "PREPRINT",
        "authors": [
            {"display_name": a, "payments": []}
            for a in (r.get("authors", "") or "").split("; ") if a
        ],
        "mesh_terms": [],
        "keywords": [],
        "journal": {"name": server, "is_predatory": False},
    }
