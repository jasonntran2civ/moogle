import type { MetadataRoute } from "next";

const BASE = process.env.NEXT_PUBLIC_SITE_URL ?? "https://evidencelens.pages.dev";

/**
 * Static-route sitemap. Per-document URLs are intentionally excluded
 * from the static sitemap because the corpus is millions of records;
 * search engines discover them via internal links and `canonical_url`
 * outlinks. If/when we want a dynamic doc sitemap, generate it from
 * BigQuery analytics.clicks (top-N most-visited) and host as a
 * sitemap index.
 */
export default function sitemap(): MetadataRoute.Sitemap {
  const now = new Date();
  return [
    { url: `${BASE}/`,           changeFrequency: "daily",   priority: 1.0,  lastModified: now },
    { url: `${BASE}/search`,     changeFrequency: "daily",   priority: 0.9,  lastModified: now },
    { url: `${BASE}/recalls`,    changeFrequency: "hourly",  priority: 0.9,  lastModified: now },
    { url: `${BASE}/about`,      changeFrequency: "monthly", priority: 0.4,  lastModified: now },
    { url: `${BASE}/docs`,       changeFrequency: "weekly",  priority: 0.5,  lastModified: now },
    { url: `${BASE}/licenses`,   changeFrequency: "monthly", priority: 0.3,  lastModified: now },
    { url: `${BASE}/changelog`,  changeFrequency: "weekly",  priority: 0.3,  lastModified: now },
  ];
}
