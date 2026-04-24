import { notFound } from "next/navigation";
import { COIBadge } from "@/components/COIBadge";
import { CitationGraph } from "@/components/CitationGraph";

async function getDoc(id: string) {
  const url = `${process.env.NEXT_PUBLIC_GATEWAY_URL}/api/document/${encodeURIComponent(id)}`;
  const res = await fetch(url, { next: { revalidate: 300 } });
  if (!res.ok) return null;
  return res.json();
}

export default async function DocumentPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const doc = await getDoc(id);
  if (!doc) notFound();

  return (
    <article className="mx-auto max-w-4xl px-4 py-8 space-y-6">
      <header className="space-y-2">
        <h1 className="text-2xl font-semibold">{doc.title}</h1>
        <p className="text-sm text-[hsl(var(--muted))]">
          {doc.journal?.name} {doc.publishedAt ? "·" : ""} {doc.publishedAt?.slice(0, 10)} · {doc.studyType}
        </p>
      </header>

      <section aria-labelledby="authors-h">
        <h2 id="authors-h" className="text-lg font-medium mb-2">Authors</h2>
        <ul className="space-y-1">
          {doc.authors?.map((a: any, i: number) => (
            <li key={i} className="flex items-center gap-2">
              <span>{a.displayName}</span>
              <COIBadge author={a} />
            </li>
          ))}
        </ul>
      </section>

      <section aria-labelledby="abstract-h">
        <h2 id="abstract-h" className="text-lg font-medium mb-2">Abstract</h2>
        <p className="whitespace-pre-line">{doc.abstract}</p>
      </section>

      <section aria-labelledby="cite-h">
        <h2 id="cite-h" className="text-lg font-medium mb-2">Citation neighborhood</h2>
        <CitationGraph documentId={doc.id} />
      </section>

      <p>
        <a className="underline" href={doc.canonicalUrl} rel="noopener noreferrer" target="_blank">
          Open original at source ↗
        </a>
      </p>
    </article>
  );
}
