import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { COIBadge } from "@/components/COIBadge";
import { CitationGraph } from "@/components/CitationGraph";

const SITE = process.env.NEXT_PUBLIC_SITE_URL ?? "https://evidencelens.pages.dev";

async function getDoc(id: string) {
  const url = `${process.env.NEXT_PUBLIC_GATEWAY_URL}/api/document/${encodeURIComponent(id)}`;
  const res = await fetch(url, { next: { revalidate: 300 } });
  if (!res.ok) return null;
  return res.json();
}

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params;
  const doc = await getDoc(id);
  if (!doc) return { title: "Not found · EvidenceLens" };
  const title = `${doc.title} · EvidenceLens`;
  const description = (doc.abstract ?? "").slice(0, 160);
  const url = `${SITE}/document/${encodeURIComponent(doc.id)}`;
  return {
    title,
    description,
    alternates: { canonical: url },
    openGraph: { title, description, url, type: "article" },
    twitter: { card: "summary_large_image", title, description },
  };
}

export default async function DocumentPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const doc = await getDoc(id);
  if (!doc) notFound();

  // JSON-LD: ScholarlyArticle with author + journal + COI annotations.
  const schemaType = doc.studyType === "TRIAL_REGISTRY" ? "MedicalStudy" : "ScholarlyArticle";
  const jsonLd: Record<string, unknown> = {
    "@context": "https://schema.org",
    "@type": schemaType,
    headline: doc.title,
    name: doc.title,
    description: (doc.abstract ?? "").slice(0, 600),
    url: doc.canonicalUrl,
    identifier: [
      ...(doc.doi   ? [{ "@type": "PropertyValue", name: "doi",   value: doc.doi }] : []),
      ...(doc.pmid  ? [{ "@type": "PropertyValue", name: "pmid",  value: doc.pmid }] : []),
      ...(doc.pmcid ? [{ "@type": "PropertyValue", name: "pmcid", value: doc.pmcid }] : []),
      ...(doc.nctId ? [{ "@type": "PropertyValue", name: "nct",   value: doc.nctId }] : []),
    ],
    datePublished: doc.publishedAt,
    license: doc.license,
    citation: doc.citationCount,
    author: (doc.authors ?? []).map((a: any) => ({
      "@type": "Person",
      name: a.displayName,
      ...(a.orcid ? { identifier: a.orcid } : {}),
      ...(a.affiliation ? { affiliation: a.affiliation } : {}),
    })),
    isPartOf: doc.journal ? { "@type": "Periodical", name: doc.journal.name, issn: doc.journal.issn } : undefined,
  };

  return (
    <article className="mx-auto max-w-4xl px-4 py-8 space-y-6">
      <script
        type="application/ld+json"
        // eslint-disable-next-line react/no-danger
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />

      <header className="space-y-2">
        <h1 className="text-2xl font-semibold">{doc.title}</h1>
        <p className="text-sm text-[hsl(var(--muted))]">
          {doc.journal?.name} {doc.publishedAt ? "·" : ""} {doc.publishedAt?.slice(0, 10)} · {doc.studyType}
        </p>
      </header>

      {doc.salience && (
        <p role="note" className="rounded border-l-4 border-[hsl(var(--accent))] bg-[hsl(var(--accent)/0.05)] p-3 text-sm">
          {doc.salience}
        </p>
      )}

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
