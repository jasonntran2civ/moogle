"use client";

import Link from "next/link";
import { forwardRef } from "react";
import { COIBadge } from "./COIBadge";

interface Props { result: any; focused?: boolean }

export const ResultCard = forwardRef<HTMLLIElement, Props>(function ResultCard({ result, focused }, ref) {
  const d = result.document;
  const url = `/document/${encodeURIComponent(d.id)}`;
  return (
    <li
      ref={ref}
      tabIndex={0}
      aria-current={focused ? "true" : undefined}
      className={
        "rounded border p-4 hover:bg-[hsl(var(--accent)/0.05)] focus:outline-none " +
        (focused ? "ring-2 ring-[hsl(var(--accent))]" : "")
      }
    >
      <Link href={url} className="text-base font-medium">
        {d.title}
      </Link>
      <div className="text-xs text-[hsl(var(--muted))] mt-0.5">
        {d.journal?.name ?? d.source} · {d.publishedAt?.slice(0, 4)} · {d.studyType}
        {typeof d.citationCount === "number" && d.citationCount > 0 && (
          <> · {d.citationCount} cites</>
        )}
      </div>
      {d.salience && (
        <p className="text-xs text-[hsl(var(--accent))] mt-1 italic">{d.salience}</p>
      )}
      {d.authors?.length ? (
        <div className="text-sm mt-1 flex flex-wrap gap-1">
          {d.authors.slice(0, 6).map((a: any, i: number) => (
            <span key={i}>
              {a.displayName}<COIBadge author={a} />{i < Math.min(5, d.authors.length - 1) ? "," : ""}
            </span>
          ))}
          {d.authors.length > 6 && <span>…</span>}
        </div>
      ) : null}
      {d.abstract && <p className="text-sm mt-2 line-clamp-3">{d.abstract}</p>}
    </li>
  );
});
