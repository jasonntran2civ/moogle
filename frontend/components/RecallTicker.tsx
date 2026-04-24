"use client";

import { useEffect, useState } from "react";

interface Recall { recallId: string; productName: string; agency: string; recallClass: string; emittedAt: string }

export function RecallTicker() {
  const [items, setItems] = useState<Recall[]>([]);
  useEffect(() => {
    fetch(`${process.env.NEXT_PUBLIC_GATEWAY_URL}/api/recalls/recent?since_days=7&top_k=5`)
      .then(r => r.ok ? r.json() : { events: [] })
      .then(data => setItems(data.events ?? []))
      .catch(() => {});
  }, []);

  if (items.length === 0) return null;
  return (
    <section aria-labelledby="recalls-h" className="rounded border bg-[hsl(var(--coi)/0.05)] p-3 text-sm">
      <h2 id="recalls-h" className="font-medium mb-1">Recent recalls (last 7 days)</h2>
      <ul className="space-y-1">
        {items.map(r => (
          <li key={r.recallId}>
            <strong>{r.productName}</strong> — {r.agency.toUpperCase()} class {r.recallClass}
          </li>
        ))}
      </ul>
    </section>
  );
}
