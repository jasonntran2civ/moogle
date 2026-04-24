import Link from "next/link";
import { SearchInput } from "@/components/SearchInput";
import { RecallTicker } from "@/components/RecallTicker";

export default function Home() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:py-24 space-y-10">
      <header className="text-center space-y-3">
        <h1 className="text-4xl sm:text-5xl font-semibold tracking-tight">EvidenceLens</h1>
        <p className="text-base text-[hsl(var(--muted))]">
          Free, public, agentic biomedical evidence search.
        </p>
      </header>

      <SearchInput placeholder="Search papers, trials, recalls — try 'sglt2 inhibitors heart failure'" autoFocus />

      <RecallTicker />

      <nav aria-label="Top-level" className="flex flex-wrap justify-center gap-x-6 gap-y-2 text-sm">
        <Link href="/about">About</Link>
        <Link href="/recalls">Recent recalls</Link>
        <Link href="/docs">Docs</Link>
        <Link href="/licenses">Licenses</Link>
        <Link href="/changelog">Changelog</Link>
      </nav>
    </div>
  );
}
