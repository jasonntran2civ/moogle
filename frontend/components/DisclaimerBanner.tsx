export function DisclaimerBanner() {
  return (
    <div role="region" aria-label="Disclaimer" className="bg-[hsl(var(--coi)/0.1)] border-b border-[hsl(var(--coi)/0.3)] text-sm">
      <p className="mx-auto max-w-7xl px-4 py-2">
        EvidenceLens is a research tool — <strong>not medical advice</strong>.
        COI badges are computed from public records via fuzzy matching and may contain false positives.
        Always verify against primary sources.
      </p>
    </div>
  );
}
