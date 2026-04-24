import { create } from "zustand";
import { persist } from "zustand/middleware";

interface SearchFilters {
  studyTypes?: string[];
  publishedYearMin?: number;
  publishedYearMax?: number;
  onlyWithCoi?: boolean;
  onlyWithFullText?: boolean;
  excludePredatoryJournals?: boolean;
}

interface SearchState {
  query: string;
  filters: SearchFilters;
  setQuery: (q: string) => void;
  toggleStudyType: (t: string) => void;
  setFilter: <K extends keyof SearchFilters>(k: K, v: SearchFilters[K]) => void;
}

export const useSearchStore = create<SearchState>()((set) => ({
  query: "",
  filters: {},
  setQuery: (q) => set({ query: q }),
  toggleStudyType: (t) => set((s) => {
    const cur = s.filters.studyTypes ?? [];
    const next = cur.includes(t) ? cur.filter(x => x !== t) : [...cur, t];
    return { filters: { ...s.filters, studyTypes: next.length ? next : undefined } };
  }),
  setFilter: (k, v) => set((s) => ({ filters: { ...s.filters, [k]: v } })),
}));

interface ByokState {
  tier: "byok" | "mcp" | "webllm";
  provider: "anthropic" | "openai" | "groq";
  key: string;
  setTier: (t: ByokState["tier"]) => void;
  setProvider: (p: ByokState["provider"]) => void;
  setKey: (k: string) => void;
}

export const useByokStore = create<ByokState>()(
  persist(
    (set) => ({
      tier: "byok",
      provider: "anthropic",
      key: "",
      setTier: (t) => set({ tier: t }),
      setProvider: (p) => set({ provider: p }),
      setKey: (k) => set({ key: k }),
    }),
    { name: "evidencelens-byok" },
  ),
);
