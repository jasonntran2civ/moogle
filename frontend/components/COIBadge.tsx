"use client";

import { useState } from "react";

interface Author {
  displayName: string;
  badge?: {
    hasPayments: boolean;
    totalPaymentsUsd: number;
    topSponsor?: string;
    topSponsorAmountUsd?: number;
    paymentsLastYear?: number;
    yearsCovered?: string[];
  };
  payments?: any[];
}

/**
 * Flagship component. Renders next to author names. Click/hover reveals
 * a tooltip with sponsor + amount detail. Conservative rendering: when
 * `hasPayments` is false (no Open Payments match above threshold), we
 * render nothing — never a "no payments" badge, since absence of match
 * is not absence of conflicts.
 */
export function COIBadge({ author }: { author: Author }) {
  const [open, setOpen] = useState(false);
  const b = author.badge;
  if (!b?.hasPayments) return null;
  const fmt = (n: number) => "$" + n.toLocaleString("en-US", { maximumFractionDigits: 0 });

  return (
    <span className="relative inline-block">
      <button
        type="button"
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        onClick={() => setOpen((v) => !v)}
        aria-label={`COI for ${author.displayName}: ${fmt(b.totalPaymentsUsd)} from ${b.topSponsor}`}
        aria-expanded={open}
        className="rounded bg-[hsl(var(--coi)/0.15)] text-[hsl(var(--coi))] text-xs font-medium px-1.5 py-0.5 ml-1"
      >
        COI {fmt(b.totalPaymentsUsd)}
      </button>
      {open && (
        <span
          role="tooltip"
          className="absolute left-0 top-full mt-1 z-50 w-64 rounded-md border bg-white dark:bg-zinc-900 p-2 text-sm shadow-lg"
        >
          <strong className="block">{author.displayName}</strong>
          <span className="block text-[hsl(var(--muted))]">
            {fmt(b.totalPaymentsUsd)} total · top {b.topSponsor} {b.topSponsorAmountUsd ? fmt(b.topSponsorAmountUsd) : ""}
          </span>
          {b.yearsCovered?.length ? (
            <span className="block text-xs">Years: {b.yearsCovered.join(", ")}</span>
          ) : null}
          <span className="block text-xs mt-1">From CMS Open Payments via fuzzy match (≥0.90 confidence).</span>
        </span>
      )}
    </span>
  );
}
