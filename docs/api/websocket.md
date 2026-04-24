# WebSocket Catalog (`/ws`)

The query-gateway exposes a single WebSocket endpoint at `/ws`. Clients send
JSON request frames and receive zero-or-more JSON response frames per
request. This document is the public message catalog and is part of the
contracts freeze (any change requires `rfc-interface` PR per
[docs/rfcs/README.md](../rfcs/README.md)).

**Spec source of truth:** [`docs/EVIDENCELENS_SPEC.md`](../EVIDENCELENS_SPEC.md) §5.6.

## Connection

- URL: `wss://gateway-evidencelens.<account>.workers.dev/ws`
- Subprotocol: `evidencelens.v1`
- Heartbeat: client sends `{"type":"ping"}` every 25s; server replies `{"type":"pong"}`. Connection drops on 60s silence.
- Rate limit: 10 simultaneous connections per IP (spec §5.6).
- Auth: none. Anonymous session ids only.

## Frame envelope

Every frame is one JSON object with at minimum a `type` field.

```jsonc
{ "type": "<message-type>", "id": "<client-correlation-id>", ...payload }
```

The `id` is opaque to the server but echoed in every response so clients
can correlate streamed responses.

## Client → server

### `search`

Request a streamed search. The server replies with one or more
`search.partial` frames followed by exactly one `search.final`.

```jsonc
{
  "type": "search",
  "id": "q-abc123",
  "query": "sglt2 inhibitors heart failure",
  "filters": { /* SearchFiltersInput shape, see graphql schema */ },
  "topK": 50,
  "variant": "rrf_k60",          // optional
  "sessionId": "anon-xyz"        // optional
}
```

### `getDoc`

Request one document. Server replies with one `getDoc.result` frame.

```jsonc
{ "type": "getDoc", "id": "d-7", "documentId": "pubmed:12345678" }
```

### `subscribe`

Subscribe to a server-pushed stream. Repeats indefinitely until
`unsubscribe`. Currently supported topics: `recalls`.

```jsonc
{
  "type": "subscribe",
  "id": "sub-1",
  "topic": "recalls",
  "filters": { "drugClass": "SGLT2", "productName": "Jardiance" }
}
```

### `unsubscribe`

```jsonc
{ "type": "unsubscribe", "id": "sub-1" }
```

### `ping` / `pong`

Heartbeat. See Connection.

## Server → client

### `search.partial`

One wave of search results. Three waves per query (5 + 10 + 35).

```jsonc
{
  "type": "search.partial",
  "id": "q-abc123",
  "wave": 1,
  "isFinal": false,
  "elapsedMs": 198,
  "results": [
    {
      "document": { /* Document shape */ },
      "finalScore": 12.31,
      "breakdown": { "bm25": 9.0, "vector": 0.83, ... }
    }
  ]
}
```

### `search.final`

The terminal frame for a `search` request. May or may not contain results
(an empty `results` array means the prior partial frames were complete).

```jsonc
{
  "type": "search.final",
  "id": "q-abc123",
  "wave": 3,
  "isFinal": true,
  "elapsedMs": 940,
  "totalEstimated": 1183,
  "results": [ /* ... */ ]
}
```

### `getDoc.result`

```jsonc
{
  "type": "getDoc.result",
  "id": "d-7",
  "document": { /* Document shape */ }
}
```

### `recall.fanout`

Server-pushed recall event for `subscribe topic=recalls`. SLO: emitted
within 1 minute of upstream FDA publication (spec §14.1).

```jsonc
{
  "type": "recall.fanout",
  "id": "sub-1",
  "event": {
    "recallId": "fda-recall:F-2026-00123",
    "agency": "fda",
    "productName": "Sample Drug 50mg",
    "drugClass": "SGLT2",
    "recallClass": "II",
    "emittedAt": "2026-04-24T15:30:00Z"
  }
}
```

### `error`

Returned when a frame is malformed, references an unknown id, or the
server hits a non-fatal failure. `code` is one of: `bad_request`,
`unknown_topic`, `not_found`, `rate_limited`, `internal_error`.

```jsonc
{
  "type": "error",
  "id": "q-abc123",
  "code": "rate_limited",
  "message": "60 req/min limit exceeded; retry after 23s"
}
```

## Client implementation notes

- Render every `search.partial` immediately and replace prior content;
  don't wait for `search.final`. The frontend's `ResultsStream` ARIA
  live region announces each wave.
- On reconnect, a session may resume an in-flight query by re-sending
  `search` with the same `id`; the server returns a fresh stream
  (idempotent rerun).
- `subscribe` survives reconnect only if the client re-sends the same
  message — there is no server-side session state.
