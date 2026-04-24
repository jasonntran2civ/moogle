"""Sliding-window chunker for vector embedding inputs (spec §5.2 step 4).

Uses tiktoken (cl100k_base) as a tokenization proxy — close enough to
BGE-M3's tokenizer for chunk-size budgeting without adding a heavy
transformer dependency to the chunker hot path.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable

import tiktoken

_ENC = tiktoken.get_encoding("cl100k_base")


@dataclass
class Chunk:
    index: int
    text: str
    token_count: int


def chunk(text: str, target_tokens: int = 512, overlap: int = 64) -> list[Chunk]:
    """Sliding window. Returns at least one chunk even for empty input."""
    if not text:
        return [Chunk(0, "", 0)]
    ids = _ENC.encode(text)
    if len(ids) <= target_tokens:
        return [Chunk(0, text, len(ids))]
    out: list[Chunk] = []
    step = target_tokens - overlap
    if step <= 0:
        step = target_tokens
    idx = 0
    cursor = 0
    while cursor < len(ids):
        window = ids[cursor:cursor + target_tokens]
        out.append(Chunk(idx, _ENC.decode(window), len(window)))
        idx += 1
        cursor += step
    return out


def mean_pool(vectors: Iterable[list[float]]) -> list[float]:
    """Per-document vector = mean of per-chunk vectors."""
    arr = list(vectors)
    if not arr:
        return []
    dim = len(arr[0])
    out = [0.0] * dim
    for v in arr:
        for i, x in enumerate(v):
            out[i] += x
    n = float(len(arr))
    return [x / n for x in out]
