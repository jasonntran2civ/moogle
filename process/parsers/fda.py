"""openFDA JSON parser (drug+device sub-endpoints). Stub."""
from __future__ import annotations

import json
from typing import Any


def parse(raw: bytes) -> dict[str, Any]:
    r = json.loads(raw)
    # The processor sets `source` from the RawDocEvent attributes; here we
    # produce shape only.
    rid = r.get("recall_number") or r.get("application_number") or r.get("k_number") or r.get("report_number") or ""
    title = r.get("product_description") or r.get("openfda", {}).get("brand_name", [""])[0] or rid
    return {
        "id": f"openfda:{rid}",
        "source": "openfda",
        "source_native_id": rid,
        "title": str(title),
        "abstract": r.get("reason_for_recall", "") or r.get("indications_and_usage", "") or "",
        "canonical_url": f"https://api.fda.gov/drug/enforcement.json?search=recall_number:{rid}",
        "license": "public-domain",
        "study_type": "REGULATORY",
        "regulatory": {
            "agency": "fda",
            "event_type": _event_type_for(r),
            "product_name": str(title),
            "drug_class": (r.get("openfda", {}).get("pharm_class_epc") or [None])[0],
            "recall_class": r.get("classification"),
            "reason": r.get("reason_for_recall"),
            "action": r.get("voluntary_mandated"),
        },
        "authors": [],
        "mesh_terms": [],
        "keywords": [],
    }


def _event_type_for(r: dict) -> str:
    if "recall_number" in r:
        return "recall"
    if "application_number" in r:
        return "approval"
    if "k_number" in r:
        return "510k_clearance"
    if "report_number" in r:
        return "adverse_event"
    return "approval"
