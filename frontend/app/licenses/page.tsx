export default function LicensesPage() {
  return (
    <article className="mx-auto max-w-3xl px-4 py-8 prose">
      <h1>Licenses</h1>
      <p>EvidenceLens itself is MIT. Per-source data licenses:</p>
      <ul>
        <li>PubMed, openFDA, ClinicalTrials.gov, NIH RePORTER, NSF Awards: public domain.</li>
        <li>OpenAlex, Unpaywall: CC0.</li>
        <li>bioRxiv: CC-BY (per preprint).</li>
        <li>medRxiv: CC-BY-NC-ND (per preprint, default).</li>
        <li>Cochrane: per review (academic only — metadata + abstract only).</li>
        <li>NICE: OGL UK.</li>
      </ul>
      <p>See <a href="/docs/sources">docs/sources/</a> for full attribution per source.</p>
    </article>
  );
}
