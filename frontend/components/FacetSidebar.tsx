"use client";

import { useEffect, useState } from "react";
import { useSearchStore } from "@/lib/store";

const STUDY_TYPES = ["RCT", "META_ANALYSIS", "SYSTEMATIC_REVIEW", "OBSERVATIONAL", "PREPRINT", "REGULATORY", "GUIDELINE"];
const SORT_MODES = [
  { id: "relevance",        label: "Relevance" },
  { id: "most_recent",      label: "Most recent" },
  { id: "most_cited",       label: "Most cited" },
  { id: "most_influential", label: "Most influential" },
];

export function FacetSidebar() {
  const { filters, toggleStudyType, setFilter } = useSearchStore();
  const [open, setOpen] = useState(true);

  // Keyboard shortcut `f` (handled in ResultsStream) dispatches this event.
  useEffect(() => {
    function onToggle() { setOpen(v => !v); }
    window.addEventListener("evidencelens:toggle-facets", onToggle as EventListener);
    return () => window.removeEventListener("evidencelens:toggle-facets", onToggle as EventListener);
  }, []);

  if (!open) {
    return (
      <aside aria-label="Filters" className="text-sm">
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="text-[hsl(var(--accent))] underline"
        >
          Show filters (f)
        </button>
      </aside>
    );
  }

  return (
    <aside aria-label="Filters" className="space-y-4 text-sm">
      <div className="flex items-center justify-between">
        <h2 className="font-medium">Filters</h2>
        <button
          type="button"
          onClick={() => setOpen(false)}
          aria-label="Hide filters (f)"
          className="text-xs text-[hsl(var(--muted))]"
        >
          hide (f)
        </button>
      </div>

      <fieldset>
        <legend className="font-medium mb-1">Sort by</legend>
        <select
          value={filters.sortMode ?? "relevance"}
          onChange={(e) => setFilter("sortMode", e.target.value)}
          className="w-full border rounded px-1 py-1"
        >
          {SORT_MODES.map(m => <option key={m.id} value={m.id}>{m.label}</option>)}
        </select>
      </fieldset>

      <fieldset>
        <legend className="font-medium mb-1">Study type</legend>
        {STUDY_TYPES.map(s => (
          <label key={s} className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={filters.studyTypes?.includes(s) ?? false}
              onChange={() => toggleStudyType(s)}
            />
            {s.replaceAll("_", " ").toLowerCase()}
          </label>
        ))}
      </fieldset>

      <fieldset>
        <legend className="font-medium mb-1">Year</legend>
        <div className="flex items-center gap-2">
          <input
            type="number" placeholder="from" min="1900" max="2100"
            className="w-20 border rounded px-1"
            value={filters.publishedYearMin ?? ""}
            onChange={(e) => setFilter("publishedYearMin", e.target.value ? parseInt(e.target.value, 10) : undefined)}
          />
          <input
            type="number" placeholder="to" min="1900" max="2100"
            className="w-20 border rounded px-1"
            value={filters.publishedYearMax ?? ""}
            onChange={(e) => setFilter("publishedYearMax", e.target.value ? parseInt(e.target.value, 10) : undefined)}
          />
        </div>
      </fieldset>

      <fieldset>
        <legend className="font-medium mb-1">Quality</legend>
        <label className="flex items-center gap-2">
          <input
            type="checkbox" checked={!!filters.onlyWithFullText}
            onChange={(e) => setFilter("onlyWithFullText", e.target.checked)}
          />
          full text available
        </label>
        <label className="flex items-center gap-2">
          <input
            type="checkbox" checked={!!filters.excludePredatoryJournals}
            onChange={(e) => setFilter("excludePredatoryJournals", e.target.checked)}
          />
          exclude predatory journals
        </label>
        <label className="flex items-center gap-2">
          <input
            type="checkbox" checked={!!filters.onlyWithCoi}
            onChange={(e) => setFilter("onlyWithCoi", e.target.checked)}
          />
          only with COI
        </label>
      </fieldset>
    </aside>
  );
}
