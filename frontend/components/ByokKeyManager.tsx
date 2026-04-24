"use client";

import { useByokStore } from "@/lib/store";

const PROVIDERS = [
  { id: "anthropic", label: "Anthropic", url: "https://console.anthropic.com/settings/keys" },
  { id: "openai",    label: "OpenAI",    url: "https://platform.openai.com/api-keys" },
  { id: "groq",      label: "Groq",      url: "https://console.groq.com/keys" },
];

export function ByokKeyManager() {
  const { provider, key, setProvider, setKey } = useByokStore();
  return (
    <div className="space-y-2 text-sm">
      <label className="block">
        Provider
        <select
          value={provider} onChange={(e) => setProvider(e.target.value as any)}
          className="ml-2 border rounded px-2 py-1"
        >
          {PROVIDERS.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
        </select>
      </label>
      <label className="block">
        API key
        <input
          type="password" value={key} onChange={(e) => setKey(e.target.value)}
          placeholder="sk-..." autoComplete="off"
          className="block w-full border rounded px-2 py-1"
        />
      </label>
      <p className="text-xs text-[hsl(var(--muted))]">
        Your key is stored in this browser only (localStorage). EvidenceLens never sees or stores it.
        It is sent only to your chosen provider via our LLM proxy. Get a key from{" "}
        <a className="underline" href={PROVIDERS.find(p => p.id === provider)?.url} rel="noopener noreferrer" target="_blank">
          {provider}
        </a>.
      </p>
    </div>
  );
}
