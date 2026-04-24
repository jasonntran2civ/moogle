"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
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
 * Spec §8.4 keyboard navigation:
 *   j / k     move focus between results
 *   Enter     open the focused result (router.push to /document/[id])
 *   Esc       blur focused result
 *   /         focus search input (handled in SearchInput)
 *   f         dispatch CustomEvent('evidencelens:toggle-facets') so
 *             FacetSidebar can listen and toggle without a context
 *   ?         handled by KeyboardHelp
 */
export function ResultsStream({ query }: { query: string }) {
  const router = useRouter();
  const [results, setResults] = useState<Result[]>([]);
  const [done, setDone] = useState(false);
  const [focused, setFocused] = useState<number>(-1);
  const itemRefs = useRef<Array<HTMLLIElement | null>>([]);

  useEffect(() => {
    if (!query) return;
    setResults([]);
    setDone(false);
    setFocused(-1);
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
          setResults(prev => (f.wave === 1 ? f.results : [...prev, ...f.results]));
          if (f.isFinal) setDone(true);
        }
      } catch { /* ignore */ }
    };
    ws.onerror = () => setDone(true);
    return () => ws.close();
  }, [query]);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      const tag = (document.activeElement?.tagName ?? "").toLowerCase();
      const editing = tag === "input" || tag === "textarea" || (document.activeElement as any)?.isContentEditable;
      if (editing) return;
      if (e.key === "j") {
        e.preventDefault();
        setFocused(i => {
          const next = Math.min(results.length - 1, i + 1);
          itemRefs.current[next]?.focus();
          return next;
        });
      } else if (e.key === "k") {
        e.preventDefault();
        setFocused(i => {
          const next = Math.max(0, i - 1);
          itemRefs.current[next]?.focus();
          return next;
        });
      } else if (e.key === "Enter" && focused >= 0 && results[focused]) {
        e.preventDefault();
        const id = results[focused].document?.id;
        if (id) router.push(`/document/${encodeURIComponent(id)}`);
      } else if (e.key === "Escape") {
        if (focused >= 0) {
          itemRefs.current[focused]?.blur();
          setFocused(-1);
        }
      } else if (e.key === "f") {
        e.preventDefault();
        window.dispatchEvent(new CustomEvent("evidencelens:toggle-facets"));
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [results, focused, router]);

  if (!query) return <p className="text-[hsl(var(--muted))]">Type a query above to begin.</p>;

  return (
    <div role="region" aria-label="Search results" aria-busy={!done}>
      <div role="status" aria-live="polite" className="sr-only">
        {done ? `${results.length} results loaded` : `Loading wave ${results.length === 0 ? 1 : ""}…`}
      </div>
      <ul className="space-y-3">
        {results.map((r, i) => (
          <ResultCard
            key={r.document.id ?? i}
            result={r}
            focused={i === focused}
            ref={(el) => { itemRefs.current[i] = el; }}
          />
        ))}
      </ul>
      {results.length === 0 && done && (
        <p>No results. Try broader terms or adjust facet filters.</p>
      )}
    </div>
  );
}
