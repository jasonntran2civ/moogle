"use client";

import { useState } from "react";

/**
 * WebLLM setup: lazy-load the @mlc-ai/web-llm bundle on first activation
 * so the ~2GB Llama 3.2 3B weights aren't fetched until the user opts in.
 *
 * Manual tool-use loop: WebLLM has no native tool-calling; we parse JSON
 * objects from model output and dispatch them as tool calls to the
 * gateway, then re-prompt with the result.
 */
export function WebLLMSetup() {
  const [progress, setProgress] = useState<string>("Not loaded.");
  const [loaded, setLoaded] = useState(false);

  async function load() {
    setProgress("Initializing WebLLM (one-time ~2GB download)…");
    try {
      const { CreateMLCEngine } = await import("@mlc-ai/web-llm");
      const engine = await CreateMLCEngine("Llama-3.2-3B-Instruct-q4f16_1-MLC", {
        initProgressCallback: (r: any) => setProgress(`${(r.progress * 100).toFixed(0)}% — ${r.text}`),
      });
      (window as any).__evidencelens_webllm = engine;
      setLoaded(true);
      setProgress("Ready.");
    } catch (e) {
      setProgress(`Failed: ${(e as Error).message}`);
    }
  }

  return (
    <div className="space-y-2 text-sm">
      <button
        type="button" onClick={load} disabled={loaded}
        className="rounded bg-[hsl(var(--accent))] text-white px-3 py-1 disabled:opacity-50"
      >
        {loaded ? "Loaded" : "Download model & start"}
      </button>
      <p role="status" aria-live="polite" className="text-xs text-[hsl(var(--muted))]">{progress}</p>
      <p className="text-xs text-[hsl(var(--muted))]">
        Model weights are served from Cloudflare R2 (no egress fee). They cache in your browser after first load.
      </p>
    </div>
  );
}
