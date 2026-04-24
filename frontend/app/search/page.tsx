"use client";

import { Suspense, useEffect } from "react";
import { useSearchParams } from "next/navigation";
import { SearchInput } from "@/components/SearchInput";
import { ResultsStream } from "@/components/ResultsStream";
import { FacetSidebar } from "@/components/FacetSidebar";
import { TierPicker } from "@/components/TierPicker";
import { useSearchStore } from "@/lib/store";

function SearchPageBody() {
  const params = useSearchParams();
  const q = params.get("q") ?? "";
  const setQuery = useSearchStore(s => s.setQuery);

  useEffect(() => { setQuery(q); }, [q, setQuery]);

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 grid grid-cols-1 lg:grid-cols-[280px_minmax(0,1fr)_320px] gap-6">
      <FacetSidebar />
      <div className="space-y-6">
        <SearchInput initialValue={q} />
        <ResultsStream query={q} />
      </div>
      <aside aria-label="Tier picker" className="space-y-4">
        <TierPicker />
      </aside>
    </div>
  );
}

export default function SearchPage() {
  return (
    <Suspense fallback={<div role="status" aria-live="polite">Loading…</div>}>
      <SearchPageBody />
    </Suspense>
  );
}
