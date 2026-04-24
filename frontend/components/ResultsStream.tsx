"use client";

import { useEffect, useState } from "react";
import { ResultCard } from "./ResultCard";

interface Result {
  document: any;
  finalScore: number;
  breakdown: any;
}

/**
 * Subscribes to /ws and renders streamed search.partial / search.final
 * frames into an ARIA live region for screen-reader announcements as
 * each wave arrives.
 *
 * j/k/Enter/Esc keyboard navigation per spec section 8.
 */
export function ResultsStream({ query }: { query: string }) {
  const [results, setResults] = useState<Result[]>([]);
  const [done, setDone] = useState(false);
  const [focused, setFocused] = useState<number>(-1);

  useEffect(() => {
    if (!query) return;
    setResults([]);
    setDone(false);
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080/ws";
    const ws = new WebSocket(wsUrl, ["evidencelens.v1"]);
    const id = `q-${Date.now()}`;

    ws.onopen = () => {
      ws.send(JSON.stringify({ type: "search", id, query, topK: 50 }));
    };
    ws.onmessage = (e) => {
      try {
        const f = JSON.parse(e.data);
        if (f.id !== id) return;
        if (f.type === "search.partial" || f.type === "search.final") {
          setResults(prev => f.wave === 1 ? f.results : [...prev, ...f.results]);
          if (f.isFinal) setDone(true);
        }
      } catch { /* ignore */ }
    };
    ws.onerror = () => setDone(true);
    return () => ws.close();
  }, [query]);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (document.activeElement?.tagName === "INPUT") return;
      if (e.key === "j") setFocused(i => Math.min(results.length - 1, i + 1));
      if (e.key === "k") setFocused(i => Math.max(0, i - 1));
      if (e.key === "Escape") setFocused(-1);
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [results.length]);

  if (!query) return <p className="text-[hsl(var(--muted))]">Type a query above to begin.</p>;

  return (
    <div role="region" aria-label="Search results" aria-busy={!done}>
      <div role="status" aria-live="polite" className="sr-only">
        {done ? `${results.length} results loaded` : `Loading wave ${results.length === 0 ? 1 : ""}…`}
      </div>
      <ul className="space-y-3">
        {results.map((r, i) => (
          <ResultCard key={r.document.id ?? i} result={r} focused={i === focused} />
        ))}
      </ul>
      {results.length === 0 && done && (
        <p>No results. Try broader terms or adjust facet filters.</p>
      )}
    </div>
  );
}
