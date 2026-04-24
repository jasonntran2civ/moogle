/**
 * click-logger Cloudflare Worker (spec §5.10).
 *
 * Receives POST /events from the frontend with one or more click events,
 * validates the schema, batches, and forwards to the pubsub-bridge
 * service binding which handles Pub/Sub publish to `click-events`.
 *
 * Free tier hard cap: 100k req/day. Per-IP rate limit (60/min) backed
 * by KV to bound abuse without needing a paid Workers Rate Limiting
 * binding.
 */
export interface Env {
  BRIDGE: Fetcher;
  RATE_LIMIT: KVNamespace;
  PUBSUB_TOPIC: string;
  GCP_PROJECT: string;
}

interface ClickEvent {
  event_id: string;
  session_id: string;
  query_id: string;
  query_text?: string;
  variant?: string;
  clicked_doc_id: string;
  clicked_position: number;
  result_set_size?: number;
  facets?: Record<string, unknown>;
  client_ts: string;
}

function validate(e: unknown): e is ClickEvent {
  if (typeof e !== "object" || e === null) return false;
  const o = e as any;
  return typeof o.event_id === "string"
    && typeof o.session_id === "string"
    && typeof o.query_id === "string"
    && typeof o.clicked_doc_id === "string"
    && typeof o.clicked_position === "number"
    && typeof o.client_ts === "string";
}

export default {
  async fetch(req: Request, env: Env, _ctx: ExecutionContext): Promise<Response> {
    if (req.method === "OPTIONS") return cors(new Response(null, { status: 204 }));
    if (req.method !== "POST") return new Response("POST only", { status: 405 });

    const url = new URL(req.url);
    if (url.pathname !== "/events") return new Response("not found", { status: 404 });

    const ip = req.headers.get("cf-connecting-ip") ?? "unknown";
    if (!await checkRateLimit(env.RATE_LIMIT, ip)) {
      return new Response("rate limited", { status: 429, headers: { "Retry-After": "60" } });
    }

    const body = await req.json().catch(() => null);
    if (!Array.isArray(body)) return new Response("expected array", { status: 400 });
    const validEvents = (body as unknown[]).filter(validate) as ClickEvent[];
    if (validEvents.length === 0) return new Response("no valid events", { status: 400 });

    // Annotate server_ts + country from CF.
    const country = (req.cf as any)?.country ?? null;
    const enriched = validEvents.map(e => ({ ...e, server_ts: new Date().toISOString(), country }));

    // Forward to the bridge worker (which signs to GCP Pub/Sub).
    const fwd = await env.BRIDGE.fetch(`https://internal/publish/${env.PUBSUB_TOPIC}`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ messages: enriched }),
    });
    if (!fwd.ok) {
      // 503 so frontend retries; click loss is tolerable but track it.
      return new Response(`bridge ${fwd.status}`, { status: 503 });
    }
    return cors(new Response(JSON.stringify({ accepted: enriched.length }), {
      headers: { "content-type": "application/json" },
    }));
  },
} satisfies ExportedHandler<Env>;

async function checkRateLimit(kv: KVNamespace, ip: string, limit = 60): Promise<boolean> {
  const minute = Math.floor(Date.now() / 60_000);
  const key = `rl:${ip}:${minute}`;
  const cur = parseInt(await kv.get(key) ?? "0", 10);
  if (cur >= limit) return false;
  await kv.put(key, String(cur + 1), { expirationTtl: 120 });
  return true;
}

function cors(r: Response): Response {
  r.headers.set("access-control-allow-origin", "*");
  r.headers.set("access-control-allow-methods", "POST, OPTIONS");
  r.headers.set("access-control-allow-headers", "content-type");
  return r;
}
