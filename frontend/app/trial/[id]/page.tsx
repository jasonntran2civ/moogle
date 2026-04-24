import { notFound } from "next/navigation";

async function getDoc(id: string) {
  const url = `${process.env.NEXT_PUBLIC_GATEWAY_URL}/api/document/${encodeURIComponent(id)}`;
  const res = await fetch(url, { next: { revalidate: 300 } });
  if (!res.ok) return null;
  return res.json();
}

export default async function TrialPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  // Trial ids land in the canonical Document store as "nct:..." or "ictrp:...".
  const docId = id.startsWith("nct:") || id.startsWith("ictrp:") ? id : `nct:${id}`;
  const doc = await getDoc(docId);
  if (!doc?.trial) notFound();
  const t = doc.trial;
  return (
    <article className="mx-auto max-w-4xl px-4 py-8 space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold">{doc.title}</h1>
        <p className="text-sm text-[hsl(var(--muted))]">
          {t.registry?.toUpperCase()} · {t.status} · {t.phase} · enrollment {t.enrollment ?? "—"}
        </p>
      </header>
      {doc.abstract && (
        <section aria-labelledby="summary-h">
          <h2 id="summary-h" className="text-lg font-medium mb-2">Summary</h2>
          <p className="whitespace-pre-line">{doc.abstract}</p>
        </section>
      )}
      <section aria-labelledby="conditions-h" className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <h2 id="conditions-h" className="text-lg font-medium mb-2">Conditions</h2>
          <ul className="list-disc pl-5 text-sm">
            {(t.conditions ?? []).map((c: string) => <li key={c}>{c}</li>)}
          </ul>
        </div>
        <div>
          <h2 className="text-lg font-medium mb-2">Interventions</h2>
          <ul className="list-disc pl-5 text-sm">
            {(t.interventions ?? []).map((c: string) => <li key={c}>{c}</li>)}
          </ul>
        </div>
      </section>
      {t.locations?.length ? (
        <section>
          <h2 className="text-lg font-medium mb-2">Locations</h2>
          <p className="text-sm">{t.locations.slice(0, 20).join(" · ")}</p>
        </section>
      ) : null}
      {t.primary_outcome && (
        <section>
          <h2 className="text-lg font-medium mb-2">Primary outcome</h2>
          <p className="text-sm">{t.primary_outcome}</p>
        </section>
      )}
      <p>
        <a className="underline" href={doc.canonicalUrl} rel="noopener noreferrer" target="_blank">
          Open original on registry ↗
        </a>
      </p>
    </article>
  );
}
