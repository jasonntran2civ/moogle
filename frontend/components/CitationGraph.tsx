"use client";

/**
 * Citation neighborhood viz. Stub: TODO render with d3-force or visx.
 * Spec section 8: visual network graph of incoming/outgoing links.
 */
export function CitationGraph({ documentId }: { documentId: string }) {
  return (
    <div aria-label={`Citation graph for ${documentId}`} className="border rounded p-4 text-sm text-[hsl(var(--muted))]">
      Citation graph visualization (TODO d3-force)
    </div>
  );
}
