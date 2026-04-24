async function getRecent() {
  const res = await fetch(`${process.env.NEXT_PUBLIC_GATEWAY_URL}/api/recalls/recent?since_days=30`, {
    next: { revalidate: 60 },
  });
  if (!res.ok) return { events: [] };
  return res.json();
}

export default async function RecallsPage() {
  const { events } = await getRecent();
  return (
    <div className="mx-auto max-w-4xl px-4 py-8 space-y-4">
      <h1 className="text-2xl font-semibold">Recent recalls</h1>
      <p className="text-sm text-[hsl(var(--muted))]">Last 30 days, newest first.</p>
      <ul className="space-y-2">
        {events.length === 0 && <li>No recalls in the window.</li>}
        {events.map((e: any) => (
          <li key={e.recallId} className="rounded border p-3">
            <div className="font-medium">{e.productName}</div>
            <div className="text-sm">{e.agency.toUpperCase()} · class {e.recallClass} · {e.drugClass ?? "—"}</div>
            <div className="text-xs text-[hsl(var(--muted))]">{e.emittedAt}</div>
          </li>
        ))}
      </ul>
    </div>
  );
}
