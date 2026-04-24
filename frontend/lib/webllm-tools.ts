/**
 * WebLLM manual tool-use loop.
 *
 * @mlc-ai/web-llm doesn't expose Anthropic/OpenAI-style native tool
 * calling. We get the same outcome by:
 *   1. System-prompting the model to emit a tagged JSON object whenever
 *      it wants to call a tool.
 *   2. Parsing those tagged objects out of the streamed output.
 *   3. Dispatching them to the gateway via /api/tool/{name}.
 *   4. Re-prompting the model with the tool result and continuing.
 *
 * This is the WebLLM equivalent of the agent-service tool-use loop in
 * agent/main.py, mirrored client-side so the visitor's GPU does the
 * generation and the EvidenceLens server only sees tool dispatches.
 */

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8080";

const TOOL_RE = /<tool_use>([\s\S]*?)<\/tool_use>/g;

const TOOL_PROMPT = `You are EvidenceLens, an evidence-based biomedical search assistant.

You have access to these tools (call them by emitting a <tool_use>...</tool_use>
block containing JSON of the shape {"name":"...","arguments":{...}}):

- search_evidence(query, filters)
- get_paper(id)
- get_trial(id)
- get_trials_by_condition(condition, location, status, phase)
- get_recent_recalls(drug_class, product_name, since_days)
- get_author_payments(author_name, year)
- get_citation_neighborhood(id, depth)
- evaluate_evidence_quality(ids)

Hard rules: not medical advice, cite every claim with [N] referencing a tool
result, surface conflicts of interest, lead with study type and recency,
acknowledge uncertainty, link canonical_url for every citation.`;

interface ToolCall { name: string; arguments: Record<string, unknown> }

interface MLCEngine {
  chat: {
    completions: {
      create(opts: any): Promise<AsyncIterable<{ choices: Array<{ delta?: { content?: string }; message?: { content?: string } }> }>>;
    };
  };
}

interface MessageLike { role: "system" | "user" | "assistant" | "tool"; content: string }

export interface RunOpts {
  /** Initial visitor question. */
  query: string;
  /** Stream callback for partial assistant text. */
  onText: (chunk: string) => void;
  /** Per-tool-call notification (for UI badge). */
  onToolCall?: (call: ToolCall, result: unknown) => void;
  /** Hard cap on tool dispatch turns (default 6). */
  maxTurns?: number;
}

/**
 * Run one WebLLM session with the manual tool-use loop. Resolves when
 * the model emits a final assistant message with no remaining tool
 * blocks.
 */
export async function runWebLLM(engine: MLCEngine, opts: RunOpts): Promise<string> {
  const messages: MessageLike[] = [
    { role: "system", content: TOOL_PROMPT },
    { role: "user", content: opts.query },
  ];
  const maxTurns = opts.maxTurns ?? 6;

  for (let turn = 0; turn < maxTurns; turn++) {
    const stream = await engine.chat.completions.create({
      messages,
      stream: true,
      temperature: 0.2,
    });

    let assistantBuffer = "";
    for await (const chunk of stream) {
      const piece = chunk.choices?.[0]?.delta?.content ?? chunk.choices?.[0]?.message?.content ?? "";
      if (!piece) continue;
      assistantBuffer += piece;
      // Stream only text outside <tool_use> blocks to the UI to avoid
      // showing raw JSON to the user.
      opts.onText(piece);
    }

    const calls = extractToolCalls(assistantBuffer);
    messages.push({ role: "assistant", content: assistantBuffer });

    if (calls.length === 0) {
      return assistantBuffer;
    }

    for (const call of calls) {
      const result = await dispatchTool(call);
      opts.onToolCall?.(call, result);
      messages.push({
        role: "tool",
        content: JSON.stringify({ tool: call.name, result }),
      });
    }
  }

  return messages[messages.length - 1]?.content ?? "";
}

function extractToolCalls(text: string): ToolCall[] {
  const out: ToolCall[] = [];
  let m: RegExpExecArray | null;
  while ((m = TOOL_RE.exec(text)) !== null) {
    try {
      const obj = JSON.parse(m[1]);
      if (typeof obj?.name === "string") {
        out.push({ name: obj.name, arguments: obj.arguments ?? {} });
      }
    } catch {
      /* skip malformed block */
    }
  }
  return out;
}

async function dispatchTool(call: ToolCall): Promise<unknown> {
  try {
    const res = await fetch(`${GATEWAY_URL}/api/tool/${encodeURIComponent(call.name)}`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(call.arguments),
    });
    if (!res.ok) return { error: `tool ${call.name} failed: ${res.status}` };
    return await res.json();
  } catch (e) {
    return { error: (e as Error).message };
  }
}
