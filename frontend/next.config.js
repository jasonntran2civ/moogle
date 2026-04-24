/** @type {import('next').NextConfig} */
export default {
  reactStrictMode: true,
  experimental: { typedRoutes: true },
  env: {
    NEXT_PUBLIC_GATEWAY_URL: process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8080",
    NEXT_PUBLIC_WS_URL: process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080/ws",
    NEXT_PUBLIC_WEBLLM_URL: process.env.NEXT_PUBLIC_WEBLLM_URL ?? "https://webllm-shard.workers.dev",
  },
  async headers() {
    return [{
      source: "/:path*",
      headers: [
        { key: "X-Content-Type-Options",  value: "nosniff" },
        { key: "Referrer-Policy",         value: "strict-origin-when-cross-origin" },
        { key: "X-Frame-Options",         value: "DENY" },
        { key: "Content-Security-Policy", value: "default-src 'self'; img-src 'self' data: https:; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; connect-src 'self' https: wss:; worker-src 'self' blob:;" },
        { key: "Permissions-Policy",      value: "camera=(), microphone=(), geolocation=()" },
      ],
    }];
  },
};
