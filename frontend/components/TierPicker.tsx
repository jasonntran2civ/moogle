"use client";

import { useEffect, useState } from "react";
import { useByokStore } from "@/lib/store";
import { ByokKeyManager } from "./ByokKeyManager";
import { WebLLMSetup } from "./WebLLMSetup";

type Tier = "byok" | "mcp" | "webllm";

export function TierPicker() {
  const tier = useByokStore(s => s.tier);
  const setTier = useByokStore(s => s.setTier);

  return (
    <section aria-labelledby="tier-h" className="rounded border p-3 space-y-3">
      <h2 id="tier-h" className="font-medium">Choose how to synthesize answers</h2>
      <div role="radiogroup" aria-label="Inference tier" className="flex flex-col gap-2 text-sm">
        {(["byok", "mcp", "webllm"] as Tier[]).map(t => (
          <label key={t} className="flex items-start gap-2">
            <input type="radio" name="tier" value={t} checked={tier === t} onChange={() => setTier(t)} />
            <span>
              <strong>{t === "byok" ? "Bring Your Own Key" : t === "mcp" ? "Model Context Protocol" : "In-browser (WebLLM)"}</strong>
              <br />
              <span className="text-[hsl(var(--muted))] text-xs">
                {t === "byok" && "Paste your Anthropic / OpenAI / Groq key. Stored in your browser only."}
                {t === "mcp" && "Connect Claude Desktop / Cursor / Cline to mcp.evidencelens.app."}
                {t === "webllm" && "Llama 3.2 3B runs on your GPU. ~2GB download once."}
              </span>
            </span>
          </label>
        ))}
      </div>
      {tier === "byok" && <ByokKeyManager />}
      {tier === "webllm" && <WebLLMSetup />}
    </section>
  );
}
