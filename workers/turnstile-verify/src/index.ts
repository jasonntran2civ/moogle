/**
 * turnstile-verify Cloudflare Worker (spec §5.10).
 *
 * Validates a Turnstile token issued to the frontend before allowing an
 * LLM proxy request. Called by the frontend right before posting to the
 * gateway's /llm/synthesize.
 */
export interface Env { TURNSTILE_SECRET: string }

export default {
  async fetch(req: Request, env: Env): Promise<Response> {
    if (req.method === "OPTIONS") return cors(new Response(null, { status: 204 }));
    if (req.method !== "POST") return new Response("POST only", { status: 405 });
    const body = await req.json().catch(() => null) as { token?: string } | null;
    if (!body?.token) return new Response("missing token", { status: 400 });

    const ip = req.headers.get("cf-connecting-ip") ?? undefined;
    const form = new URLSearchParams();
    form.set("secret", env.TURNSTILE_SECRET);
    form.set("response", body.token);
    if (ip) form.set("remoteip", ip);

    const r = await fetch("https://challenges.cloudflare.com/turnstile/v0/siteverify", {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: form,
    });
    const data = await r.json() as { success: boolean };
    return cors(new Response(JSON.stringify({ ok: data.success }), {
      status: data.success ? 200 : 401,
      headers: { "content-type": "application/json" },
    }));
  },
} satisfies ExportedHandler<Env>;

function cors(r: Response): Response {
  r.headers.set("access-control-allow-origin", "*");
  r.headers.set("access-control-allow-methods", "POST, OPTIONS");
  r.headers.set("access-control-allow-headers", "content-type");
  return r;
}
