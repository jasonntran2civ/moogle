"""ClinicalTrials.gov v2 JSON parser. Stub: extracts the fields the
canonical schema needs; full parser is a fill-out task."""
from __future__ import annotations

import json
from typing import Any


def parse(raw: bytes) -> dict[str, Any]:
    s = json.loads(raw)
    proto = s.get("protocolSection", {})
    ident = proto.get("identificationModule", {})
    nct = ident.get("nctId", "")
    title = ident.get("officialTitle") or ident.get("briefTitle") or ""
    desc = proto.get("descriptionModule", {})
    abstract = desc.get("briefSummary") or desc.get("detailedDescription") or ""
    status = proto.get("statusModule", {}).get("overallStatus", "unknown").lower()
    phase = (proto.get("designModule", {}).get("phases") or ["NA"])[0].lower().replace(" ", "_")
    conditions = proto.get("conditionsModule", {}).get("conditions", [])
    interventions = [
        i.get("name", "") for i in proto.get("armsInterventionsModule", {}).get("interventions", [])
    ]
    locations = [
        f"{loc.get('city', '')}, {loc.get('country', '')}".strip(", ")
        for loc in proto.get("contactsLocationsModule", {}).get("locations", [])
    ]

    return {
        "id": f"nct:{nct}",
        "source": "ctgov",
        "source_native_id": nct,
        "nct_id": nct,
        "title": title,
        "abstract": abstract,
        "canonical_url": f"https://clinicaltrials.gov/study/{nct}",
        "license": "public-domain",
        "study_type": "TRIAL_REGISTRY",
        "trial": {
            "registry": "ctgov",
            "status": status,
            "phase": phase,
            "conditions": conditions,
            "interventions": interventions,
            "locations": locations,
            "enrollment": (proto.get("designModule", {}).get("enrollmentInfo", {}) or {}).get("count"),
            "primary_outcome": (proto.get("outcomesModule", {}).get("primaryOutcomes") or [{}])[0].get("measure"),
        },
        "authors": [],
        "mesh_terms": [],
        "keywords": [],
    }
