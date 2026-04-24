# Cloudflare Workers

Four edge functions per spec §5.10:

| Worker | Purpose | Cap concern |
|---|---|---|
| [pubsub-bridge](pubsub-bridge/) | Sign GCP Pub/Sub access tokens; verify Google OIDC; outbound publish + inbound forward to NATS | Internal only |
| [click-logger](click-logger/) | Validate + batch frontend ClickEvents → Pub/Sub via `pubsub-bridge` service binding | 60/min/IP via KV; protects 100k/day cap |
| [webllm-shard](webllm-shard/) | Serve WebLLM model weights from R2 with `Range` + immutable cache headers | $0 egress on R2 |
| [turnstile-verify](turnstile-verify/) | Server-side validate a Turnstile token before LLM proxy call | n/a |

## Deploy

```bash
cd workers/<name>
wrangler secret put <SECRET_NAME>   # one-time per secret
wrangler deploy
```

GHA: [.github/workflows/deploy-workers.yml](../.github/workflows/deploy-workers.yml) runs the matrix on push to `main`.
