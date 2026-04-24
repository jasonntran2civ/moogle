"use client";

import { useSearchStore } from "@/lib/store";

const STUDY_TYPES = ["RCT", "META_ANALYSIS", "SYSTEMATIC_REVIEW", "OBSERVATIONAL", "PREPRINT", "REGULATORY", "GUIDELINE"];

export function FacetSidebar() {
  const { filters, toggleStudyType, setFilter } = useSearchStore();
  return (
    <aside aria-label="Filters" className="space-y-4 text-sm">
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
