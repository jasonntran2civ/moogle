/**
 * pubsub-bridge Cloudflare Worker (spec §5.10).
 *
 * Two roles:
 *   1. INBOUND: Pub/Sub push subscription POSTs here with an OIDC JWT;
 *      verify against Google JWKS, then forward to NATS over WSS.
 *   2. OUTBOUND: Other Workers (click-logger, ...) call us as a service
 *      binding to publish to GCP Pub/Sub topics. We sign a service-
 *      account JWT and POST to projects/.../topics/.../publish.
 */
export interface Env {
  GCP_PROJECT: string;
  NATS_WS_URL: string;
  GCP_SA_JSON: string;  // Workers secret
}

export default {
  async fetch(req: Request, env: Env): Promise<Response> {
    const url = new URL(req.url);

    // Outbound: /publish/{topic}
    if (url.pathname.startsWith("/publish/")) {
      const topic = decodeURIComponent(url.pathname.slice("/publish/".length));
      const body = await req.json() as { messages: unknown[] };
      const accessToken = await mintAccessToken(env.GCP_SA_JSON);
      const upstream = await fetch(
        `https://pubsub.googleapis.com/v1/projects/${env.GCP_PROJECT}/topics/${topic}:publish`,
        {
          method: "POST",
          headers: { "authorization": `Bearer ${accessToken}`, "content-type": "application/json" },
          body: JSON.stringify({
            messages: body.messages.map(m => ({ data: btoa(JSON.stringify(m)) })),
          }),
        },
      );
      return new Response(await upstream.text(), { status: upstream.status });
    }

    // Inbound: /pubsub/{subscription}
    if (url.pathname.startsWith("/pubsub/")) {
      const auth = req.headers.get("authorization") ?? "";
      const ok = await verifyGoogleOidc(auth.replace(/^Bearer /, ""));
      if (!ok) return new Response("forbidden", { status: 403 });

      const env_msg = await req.json() as { message: { data: string } };
      const decoded = JSON.parse(atob(env_msg.message.data));

      // TODO: forward to NATS over WSS. Cloudflare Workers can connect
      // to WebSocket origins via `connect:` outbound. The NATS protocol
      // requires CONNECT + PUB framing - implement once NATS WS edge
      // is configured.
      console.log("inbound pubsub", decoded);
      return new Response("ack", { status: 200 });
    }

    return new Response("not found", { status: 404 });
  },
} satisfies ExportedHandler<Env>;

// ---- helpers (stubbed; replace with real JOSE + GAuth flow) ----

async function mintAccessToken(_saJson: string): Promise<string> {
  // TODO real implementation: parse SA JSON, sign a JWT (RS256) with
  // claims iss, scope=https://www.googleapis.com/auth/pubsub,
  // exchange at oauth2.googleapis.com/token for a 1h access token.
  return "STUB_TOKEN";
}

async function verifyGoogleOidc(_jwt: string): Promise<boolean> {
  // TODO fetch https://www.googleapis.com/oauth2/v3/certs, verify JWT
  // signature, check iss=accounts.google.com or https://accounts.google.com,
  // check aud matches our pushSubscription's `oidcToken.audience`.
  return true;
}
