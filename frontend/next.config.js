/** @type {import('next').NextConfig} */
export default {
  reactStrictMode: true,
  experimental: { typedRoutes: true },
  env: {
    NEXT_PUBLIC_GATEWAY_URL: process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8080",
    NEXT_PUBLIC_WS_URL: process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080/ws",
    NEXT_PUBLIC_WEBLLM_URL: process.env.NEXT_PUBLIC_WEBLLM_URL ?? "https://webllm-shard.workers.dev",
    NEXT_PUBLIC_SITE_URL: process.env.NEXT_PUBLIC_SITE_URL ?? "https://evidencelens.pages.dev",
  },
  async headers() {
    return [{
      source: "/:path*",
      headers: [
        { key: "Strict-Transport-Security", value: "max-age=63072000; includeSubDomains; preload" },
        { key: "X-Content-Type-Options",  value: "nosniff" },
        { key: "Referrer-Policy",         value: "strict-origin-when-cross-origin" },
        { key: "X-Frame-Options",         value: "DENY" },
        // wasm-unsafe-eval is required by WebLLM (WebAssembly + WebGPU).
        { key: "Content-Security-Policy", value: "default-src 'self'; img-src 'self' data: https:; script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' https: wss: blob:; worker-src 'self' blob:; frame-ancestors 'none';" },
        { key: "Permissions-Policy",      value: "camera=(), microphone=(), geolocation=(), interest-cohort=()" },
        { key: "Cross-Origin-Opener-Policy",   value: "same-origin" },
        { key: "Cross-Origin-Embedder-Policy", value: "require-corp" },
      ],
    }];
  },
};
