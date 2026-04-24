/**
 * pubsub-bridge Cloudflare Worker (spec §5.10).
 *
 * Two roles:
 *   1. INBOUND  — Pub/Sub push subscription POSTs to /pubsub/{topic};
 *      verify Google OIDC JWT, then forward to NATS over WSS.
 *   2. OUTBOUND — Other Workers (click-logger, ...) call us as a service
 *      binding at /publish/{topic}; we sign a service-account JWT and
 *      POST to projects/.../topics/.../publish.
 */
export interface Env {
  GCP_PROJECT: string;
  GCP_SA_JSON: string; // wrangler secret put GCP_SA_JSON
  NATS_WS_URL?: string;
  NATS_TOKEN?: string;
  /** Audience expected on inbound OIDC tokens; configured on the GCP push subscription. */
  EXPECTED_AUDIENCE?: string;
  /** Optional: SA email expected on inbound OIDC tokens. */
  ALLOWED_PUSHER_EMAIL?: string;
}

interface ServiceAccountKey {
  client_email: string;
  private_key: string;
  token_uri: string;
}

// ---- access-token cache (per-isolate; ~1h TTL) ----

interface CachedToken { token: string; expiresAt: number }
let _tokenCache: CachedToken | null = null;

async function mintAccessToken(saJson: string): Promise<string> {
  const now = Math.floor(Date.now() / 1000);
  if (_tokenCache && _tokenCache.expiresAt > now + 60) return _tokenCache.token;

  const sa: ServiceAccountKey = JSON.parse(saJson);
  const header = { alg: "RS256", typ: "JWT" };
  const payload = {
    iss: sa.client_email,
    scope: "https://www.googleapis.com/auth/pubsub",
    aud: sa.token_uri,
    iat: now,
    exp: now + 3600,
  };
  const jwt = await signRS256(header, payload, sa.private_key);

  const res = await fetch(sa.token_uri, {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "urn:ietf:params:oauth:grant-type:jwt-bearer",
      assertion: jwt,
    }),
  });
  if (!res.ok) throw new Error(`google token exchange failed: ${res.status} ${await res.text()}`);
  const json = (await res.json()) as { access_token: string; expires_in: number };
  _tokenCache = { token: json.access_token, expiresAt: now + json.expires_in };
  return json.access_token;
}

// ---- inbound OIDC verification ----

interface CachedJwks { keys: Map<string, CryptoKey>; expiresAt: number }
let _jwks: CachedJwks | null = null;

async function loadGoogleJWKS(): Promise<Map<string, CryptoKey>> {
  const now = Math.floor(Date.now() / 1000);
  if (_jwks && _jwks.expiresAt > now) return _jwks.keys;

  const res = await fetch("https://www.googleapis.com/oauth2/v3/certs", {
    cf: { cacheTtl: 3600, cacheEverything: true },
  });
  const json = (await res.json()) as { keys: Array<{ kid: string; n: string; e: string; kty: string; alg: string }> };
  const keys = new Map<string, CryptoKey>();
  for (const j of json.keys) {
    const ck = await crypto.subtle.importKey(
      "jwk",
      { kty: j.kty, n: j.n, e: j.e, alg: j.alg, ext: true },
      { name: "RSASSA-PKCS1-v1_5", hash: "SHA-256" },
      false,
      ["verify"],
    );
    keys.set(j.kid, ck);
  }
  _jwks = { keys, expiresAt: now + 3600 };
  return keys;
}

async function verifyGoogleOidc(jwt: string, expectedAud?: string, allowedEmail?: string): Promise<boolean> {
  if (!jwt) return false;
  const parts = jwt.split(".");
  if (parts.length !== 3) return false;
  const [hB64, pB64, sB64] = parts;
  const header = JSON.parse(b64urlDecodeText(hB64));
  const payload = JSON.parse(b64urlDecodeText(pB64));

  const keys = await loadGoogleJWKS();
  const key = keys.get(header.kid);
  if (!key) return false;

  const sig = b64urlDecodeBytes(sB64);
  const data = new TextEncoder().encode(`${hB64}.${pB64}`);
  const ok = await crypto.subtle.verify("RSASSA-PKCS1-v1_5", key, sig, data);
  if (!ok) return false;

  const now = Math.floor(Date.now() / 1000);
  if (typeof payload.exp !== "number" || payload.exp <= now) return false;
  if (payload.iss !== "https://accounts.google.com" && payload.iss !== "accounts.google.com") return false;
  if (expectedAud && payload.aud !== expectedAud) return false;
  if (allowedEmail && payload.email !== allowedEmail) return false;
  if (allowedEmail && payload.email_verified !== true) return false;
  return true;
}

// ---- outbound: forward to NATS via WSS ----

async function forwardToNats(env: Env, subject: string, payload: unknown): Promise<void> {
  if (!env.NATS_WS_URL) {
    // No NATS bridge configured (single-environment dev). Log and drop.
    console.log("[pubsub-bridge] no NATS_WS_URL; dropping", { subject });
    return;
  }
  const ws = new WebSocket(env.NATS_WS_URL);
  await new Promise<void>((resolve, reject) => {
    ws.addEventListener("open", () => resolve(), { once: true });
    ws.addEventListener("error", () => reject(new Error("nats ws error")), { once: true });
  });
  // Minimal NATS protocol: CONNECT then PUB.
  ws.send(`CONNECT ${JSON.stringify({
    verbose: false, pedantic: false, tls_required: false,
    name: "pubsub-bridge", lang: "javascript", version: "0.1.0",
    headers: false, no_responders: false,
    auth_token: env.NATS_TOKEN ?? undefined,
  })}\r\n`);
  const body = JSON.stringify(payload);
  ws.send(`PUB ${subject} ${body.length}\r\n${body}\r\n`);
  // Allow the frame to flush, then close.
  setTimeout(() => ws.close(), 100);
}

// ---- handler ----

export default {
  async fetch(req: Request, env: Env): Promise<Response> {
    try {
      const url = new URL(req.url);

      if (url.pathname.startsWith("/publish/")) {
        const topic = decodeURIComponent(url.pathname.slice("/publish/".length));
        const body = (await req.json()) as { messages: unknown[] };
        const accessToken = await mintAccessToken(env.GCP_SA_JSON);
        const upstream = await fetch(
          `https://pubsub.googleapis.com/v1/projects/${env.GCP_PROJECT}/topics/${topic}:publish`,
          {
            method: "POST",
            headers: { "authorization": `Bearer ${accessToken}`, "content-type": "application/json" },
            body: JSON.stringify({
              messages: body.messages.map((m) => ({ data: btoa(JSON.stringify(m)) })),
            }),
          },
        );
        return new Response(await upstream.text(), { status: upstream.status });
      }

      if (url.pathname.startsWith("/pubsub/")) {
        const subject = url.pathname.slice("/pubsub/".length);
        const auth = req.headers.get("authorization") ?? "";
        const jwt = auth.replace(/^Bearer /, "");
        const ok = await verifyGoogleOidc(jwt, env.EXPECTED_AUDIENCE, env.ALLOWED_PUSHER_EMAIL);
        if (!ok) return new Response("forbidden", { status: 403 });

        const envMsg = (await req.json()) as { message: { data: string; attributes?: Record<string, string> } };
        const decoded = JSON.parse(b64urlDecodeText(envMsg.message.data));
        await forwardToNats(env, subject, decoded);
        return new Response("ack", { status: 200 });
      }

      if (url.pathname === "/healthz") {
        return new Response(JSON.stringify({ status: "ok" }), {
          headers: { "content-type": "application/json" },
        });
      }
      return new Response("not found", { status: 404 });
    } catch (e) {
      console.error("[pubsub-bridge] error", e);
      return new Response((e as Error).message, { status: 500 });
    }
  },
} satisfies ExportedHandler<Env>;

// ---- crypto helpers ----

async function signRS256(header: object, payload: object, pemKey: string): Promise<string> {
  const enc = new TextEncoder();
  const headerB64 = b64urlEncode(enc.encode(JSON.stringify(header)));
  const payloadB64 = b64urlEncode(enc.encode(JSON.stringify(payload)));
  const data = enc.encode(`${headerB64}.${payloadB64}`);

  const key = await crypto.subtle.importKey(
    "pkcs8",
    pemToPkcs8(pemKey),
    { name: "RSASSA-PKCS1-v1_5", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const sig = new Uint8Array(await crypto.subtle.sign("RSASSA-PKCS1-v1_5", key, data));
  return `${headerB64}.${payloadB64}.${b64urlEncode(sig)}`;
}

function pemToPkcs8(pem: string): ArrayBuffer {
  const b64 = pem
    .replace(/-----BEGIN [A-Z ]+-----/g, "")
    .replace(/-----END [A-Z ]+-----/g, "")
    .replace(/\s+/g, "");
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out.buffer;
}

function b64urlEncode(b: Uint8Array): string {
  let s = "";
  for (const v of b) s += String.fromCharCode(v);
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function b64urlDecodeText(s: string): string {
  return atob(s.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(s.length / 4) * 4, "="));
}

function b64urlDecodeBytes(s: string): Uint8Array {
  const bin = b64urlDecodeText(s);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}
