# gateway

Per spec §5.6. NestJS 11. Public API surface (REST + GraphQL + WebSocket + BYOK proxy).

## Modules

| Module | Routes |
|---|---|
| SearchModule | `GET /api/search`, `Query.search`, WS `search` |
| DocumentModule | `GET /api/document/:id`, `Query.document` |
| TrialsModule | `GET /api/trials` |
| RecallsModule | `GET /api/recalls/recent` |
| LlmProxyModule | `GET /llm/models`, `POST /llm/synthesize` (SSE) |
| AdminModule | `GET /admin/status` (Cloudflare Access only) |
| GatewayWebSocketModule | `/ws` upgrade endpoint |

## Rate limits

Set via `@nestjs/throttler`:
- `rest`: 60/min
- `llm`: 30/min

## BYOK proxy

`POST /llm/synthesize` requires `Authorization: Bearer <visitor-key>`, `X-Turnstile-Token`, `X-Provider`. Forwards SSE to `agent-service`. Key is **never logged**.

## Dev

```bash
pnpm install
pnpm start:dev
```

GraphQL playground at `http://localhost:8080/graphql`.

## Notes

- Resolvers and services are scaffolded with TODO markers; real gRPC clients to scorer-pool wire in once `proto/gen/typescript/` is generated.
- WebSocket `subscribe topic=recalls` will bridge a NATS subscription on `recall-fanout`.
