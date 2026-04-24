"use client";

import { useState } from "react";
import { runWebLLM } from "@/lib/webllm-tools";

/**
 * WebLLM setup: lazy-load the @mlc-ai/web-llm bundle on first activation
 * so the ~2GB Llama 3.2 3B weights aren't fetched until the user opts in.
 *
 * Manual tool-use loop lives in lib/webllm-tools.ts; this component
 * surfaces a "test" button that runs one round-trip end-to-end so
 * visitors can verify their browser + GPU + the gateway tool dispatch
 * all work before relying on it.
 */
export function WebLLMSetup() {
  const [progress, setProgress] = useState<string>("Not loaded.");
  const [loaded, setLoaded] = useState(false);
  const [engine, setEngine] = useState<any>(null);
  const [testing, setTesting] = useState(false);
  const [testOutput, setTestOutput] = useState<string>("");

  async function load() {
    setProgress("Initializing WebLLM (one-time ~2GB download)…");
    try {
      const { CreateMLCEngine } = await import("@mlc-ai/web-llm");
      const e = await CreateMLCEngine("Llama-3.2-3B-Instruct-q4f16_1-MLC", {
        initProgressCallback: (r: any) =>
          setProgress(`${(r.progress * 100).toFixed(0)}% — ${r.text}`),
      });
      (window as any).__evidencelens_webllm = e;
      setEngine(e);
      setLoaded(true);
      setProgress("Ready.");
    } catch (err) {
      setProgress(`Failed: ${(err as Error).message}`);
    }
  }

  async function test() {
    if (!engine) return;
    setTesting(true);
    setTestOutput("");
    try {
      await runWebLLM(engine, {
        query:
          "What is the most cited recent RCT for SGLT2 inhibitors in heart failure? Use the search tool.",
        onText: (chunk) => setTestOutput((prev) => prev + chunk),
      });
    } catch (e) {
      setTestOutput((prev) => prev + `\n[error] ${(e as Error).message}`);
    } finally {
      setTesting(false);
    }
  }

  return (
    <div className="space-y-2 text-sm">
      <button
        type="button"
        onClick={load}
        disabled={loaded}
        className="rounded bg-[hsl(var(--accent))] text-white px-3 py-1 disabled:opacity-50"
      >
        {loaded ? "Loaded" : "Download model & start"}
      </button>
      <p role="status" aria-live="polite" className="text-xs text-[hsl(var(--muted))]">
        {progress}
      </p>
      {loaded && (
        <>
          <button
            type="button"
            onClick={test}
            disabled={testing}
            className="rounded border border-[hsl(var(--accent))] text-[hsl(var(--accent))] px-3 py-1 disabled:opacity-50"
          >
            {testing ? "Running…" : "Run sample query"}
          </button>
          {testOutput && (
            <pre className="text-xs bg-[hsl(var(--muted)/0.1)] p-2 rounded whitespace-pre-wrap max-h-64 overflow-auto">
              {testOutput}
            </pre>
          )}
        </>
      )}
      <p className="text-xs text-[hsl(var(--muted))]">
        Model weights served from Cloudflare R2 (no egress fee), cached in your browser after first load.
      </p>
    </div>
  );
}
