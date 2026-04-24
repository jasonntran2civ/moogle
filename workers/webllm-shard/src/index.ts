/**
 * webllm-shard Cloudflare Worker (spec §5.10).
 *
 * Serves WebLLM model shards from R2 with appropriate CORS so the
 * visitor's browser can fetch ~2GB of weights without egress charges.
 * R2 has $0 egress; Workers free tier handles 100k req/day per script
 * which is plenty for the manifest + shard requests during model load.
 */
export interface Env { WEBLLM: R2Bucket }

export default {
  async fetch(req: Request, env: Env): Promise<Response> {
    if (req.method === "OPTIONS") return cors(new Response(null, { status: 204 }));
    if (req.method !== "GET" && req.method !== "HEAD") {
      return new Response("GET/HEAD only", { status: 405 });
    }
    const url = new URL(req.url);
    const key = url.pathname.replace(/^\//, "");
    if (!key) return new Response("missing key", { status: 400 });

    const obj = await env.WEBLLM.get(key, { range: req.headers });
    if (!obj) return new Response("not found", { status: 404 });

    const headers = new Headers();
    obj.writeHttpMetadata(headers);
    headers.set("etag", obj.httpEtag);
    headers.set("accept-ranges", "bytes");
    headers.set("cache-control", "public, max-age=31536000, immutable");
    return cors(new Response(obj.body, { status: 200, headers }));
  },
} satisfies ExportedHandler<Env>;

function cors(r: Response): Response {
  r.headers.set("access-control-allow-origin", "*");
  r.headers.set("access-control-allow-methods", "GET, HEAD, OPTIONS");
  r.headers.set("access-control-allow-headers", "range");
  r.headers.set("access-control-expose-headers", "content-length, content-range, etag");
  return r;
}
