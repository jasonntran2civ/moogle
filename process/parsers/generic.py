"""Generic parser for sources that don't need bespoke field extraction.

Many of the round-2 sources (CORE, ChEMBL, OMIM, HPO, DisGeNET, etc.)
ship JSON or already-flattened structured data. The generic parser:

  1. Loads the raw bytes as JSON (defaults to `{}` on failure).
  2. Picks the best-effort id, title, abstract, url with per-source
     overrides where the upstream uses non-standard field names.
  3. Returns a Document dict with the source label, study_type, and a
     license placeholder. Per-source license correctness is the
     ingester's responsibility (it should set raw["license"]).

Per-source overrides live in `_OVERRIDES`.
"""
from __future__ import annotations

import json
from typing import Any, Callable


_OVERRIDES: dict[str, dict[str, list[str]]] = {
    "core":          {"id": ["id", "doi"], "title": ["title"], "abstract": ["abstract"]},
    "chembl":        {"id": ["molecule_chembl_id"], "title": ["pref_name"], "abstract": ["full_molformula"]},
    "omim":          {"id": ["mimNumber"], "title": ["titles", "preferredTitle"], "abstract": ["text"]},
    "hpo":           {"id": ["id"], "title": ["name"], "abstract": ["def"]},
    "disgenet":      {"id": ["disease_id_of_intersection"], "title": ["disease_name"], "abstract": ["description"]},
    "cdc-wonder":    {"id": ["snapshot"], "title": ["snapshot"], "abstract": ["raw_xml"]},
    "ema":           {"id": ["Product number", "EMA product number"], "title": ["Name of medicine"], "abstract": ["Therapeutic area"]},
    "mhra":          {"id": ["id"], "title": ["title"], "abstract": ["link"]},
    "health-canada": {"id": ["drug_code"], "title": ["brand_name"], "abstract": ["descriptor"]},
    "tga":           {"id": ["GUID"], "title": ["Title"], "abstract": ["Description"]},
    "pmda":          {"id": ["GUID"], "title": ["Title"], "abstract": ["Description"]},
    "nsf":           {"id": ["id"], "title": ["title"], "abstract": ["abstractText"]},
    "drugbank":      {"id": ["DrugbankID"], "title": ["Name"], "abstract": ["Description"]},
    "pmc-oa":        {"id": ["pmcid"], "title": ["pmcid"], "abstract": ["file"]},
    "ictrp":         {"id": ["TrialID"], "title": ["Title"], "abstract": ["Conditions"]},
    "nih-reporter":  {"id": ["appl_id"], "title": ["project_title"], "abstract": ["abstract_text"]},
    "openalex":      {"id": ["id"], "title": ["display_name"], "abstract": ["abstract_inverted_index"]},
    "crossref":      {"id": ["DOI"], "title": ["title"], "abstract": ["abstract"]},
    "unpaywall":     {"id": ["doi"], "title": ["title"], "abstract": ["abstract"]},
    "open-payments": {"id": ["record_id"], "title": ["product_name"], "abstract": ["nature_of_payment"]},
    "cochrane":      {"id": ["doi"], "title": ["title"], "abstract": ["description"]},
    "guideline-uspstf": {"id": ["url"], "title": ["url"], "abstract": ["text"]},
    "guideline-nice":   {"id": ["url"], "title": ["url"], "abstract": ["text"]},
    "guideline-ahrq":   {"id": ["url"], "title": ["url"], "abstract": ["text"]},
}


def make_parser(
    source: str,
    *,
    source_label: str | None = None,
    study_type: str = "OTHER",
) -> Callable[[bytes], dict[str, Any]]:
    label = source_label or source

    def _parse(raw: bytes) -> dict[str, Any]:
        try:
            obj = json.loads(raw)
        except json.JSONDecodeError:
            obj = {"_raw": raw[:1000].decode("utf-8", errors="replace")}
        if not isinstance(obj, dict):
            obj = {"_value": obj}
        ov = _OVERRIDES.get(source, {})
        rid = _pick(obj, ov.get("id", ["id"])) or _pick(obj, ["DOI", "doi", "ID"]) or ""
        title = _pick(obj, ov.get("title", ["title", "name", "display_name"])) or rid or ""
        abstract = _pick(obj, ov.get("abstract", ["abstract", "description"])) or ""
        url = _pick(obj, ["url", "canonical_url", "link", "href"]) or ""
        license_str = _pick(obj, ["license", "License"]) or "unknown"

        return {
            "id": f"{label}:{rid}" if rid else f"{label}:{hash_str(json.dumps(obj, sort_keys=True))}",
            "source": label,
            "source_native_id": str(rid),
            "title": str(title)[:1000],
            "abstract": str(abstract)[:50_000],
            "canonical_url": str(url),
            "license": str(license_str),
            "study_type": study_type,
            "authors": [],
            "mesh_terms": [],
            "keywords": [],
        }

    return _parse


def _pick(obj: dict, keys: list[str]) -> Any:
    for k in keys:
        # Support nested keys with "a.b" notation.
        if "." in k:
            cur: Any = obj
            for part in k.split("."):
                if isinstance(cur, dict) and part in cur:
                    cur = cur[part]
                else:
                    cur = None
                    break
            if cur not in (None, "", []):
                return cur
        elif k in obj and obj[k] not in (None, "", []):
            return obj[k]
    return None


def hash_str(s: str) -> str:
    import hashlib
    return hashlib.sha1(s.encode("utf-8")).hexdigest()[:16]
