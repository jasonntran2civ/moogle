"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

interface Props {
  placeholder?: string;
  initialValue?: string;
  autoFocus?: boolean;
}

export function SearchInput({ placeholder = "Search EvidenceLens", initialValue = "", autoFocus }: Props) {
  const router = useRouter();
  const [value, setValue] = useState(initialValue);
  const ref = useRef<HTMLInputElement>(null);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "/" && document.activeElement?.tagName !== "INPUT") {
        e.preventDefault(); ref.current?.focus();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return (
    <form
      role="search"
      onSubmit={(e) => { e.preventDefault(); router.push(`/search?q=${encodeURIComponent(value)}`); }}
      className="flex items-center gap-2 rounded-2xl border bg-white dark:bg-zinc-900 p-2 focus-within:ring-2 ring-[hsl(var(--accent))]"
    >
      <label htmlFor="search-q" className="sr-only">Search query</label>
      <input
        id="search-q"
        ref={ref}
        type="search"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder={placeholder}
        autoFocus={autoFocus}
        className="flex-1 bg-transparent outline-none px-2 py-1.5"
        aria-keyshortcuts="/"
      />
      <button type="submit" className="rounded-xl bg-[hsl(var(--accent))] text-white px-3 py-1.5">
        Search
      </button>
    </form>
  );
}
