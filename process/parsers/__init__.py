"""Per-source parser modules. Each returns a normalized Document dict.

Dispatch by `source` field on the RawDocEvent. Adding a source = drop a
new module here and add a key to `PARSERS`.
"""
from __future__ import annotations

from typing import Callable

from . import pubmed as _pubmed
from . import trials as _trials
from . import fda as _fda
from . import preprint as _preprint

PARSERS: dict[str, Callable[[bytes], dict]] = {
    "pubmed":   _pubmed.parse,
    "ctgov":    _trials.parse,
    "biorxiv":  _preprint.parse,
    "medrxiv":  _preprint.parse,
    # openFDA sub-endpoints share the same parser shape:
    "openfda-drug-drugsfda":   _fda.parse,
    "openfda-drug-enforcement": _fda.parse,
    "openfda-device-event":    _fda.parse,
    "openfda-device-510k":     _fda.parse,
}


def parse(source: str, raw: bytes) -> dict:
    fn = PARSERS.get(source)
    if not fn:
        raise ValueError(f"no parser for source: {source}")
    return fn(raw)
