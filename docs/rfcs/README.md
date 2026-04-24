# RFC Process

After the **contracts freeze** at the end of Week 1 (tag `contracts-v1.0.0`), any change to inter-service contracts requires an RFC PR.

## What requires an RFC

- Anything in [`proto/evidencelens/v1/`](../../proto/evidencelens/v1/) (`document.proto`, `events.proto`, `scorer.proto`, `embedder.proto`)
- [`gateway/src/schema.graphql`](../../gateway/src/schema.graphql)
- [`docs/api/openapi.yaml`](../api/openapi.yaml)
- [`docs/api/websocket.md`](../api/websocket.md) — WebSocket message catalog
- [`mcp-server/src/tools.ts`](../../mcp-server/src/tools.ts) — MCP tool surface
- NATS topic / Pub/Sub topic catalog at [`docs/api/events.md`](../api/events.md)

## What doesn't

- Internal package additions, refactors, performance work, bug fixes
- Adding a new sub-scorer (as long as gRPC surface is unchanged)
- Adding a new ingester source (uses existing `RawDocEvent` schema)
- Adding new analytics dimensions (BigQuery schema is internal)

## Process

1. Open a PR labeled `rfc-interface` (for proto/REST/GraphQL/WS) or `rfc-feature` (for in-spec features that change semantics).
2. Add `docs/rfcs/{NNNN}-{slug}.md` describing motivation, proposed change, alternatives, migration impact, rollout plan.
3. Orchestrator (and on irreversible changes, the user) reviews.
4. On approval, merge bumps the contract version and updates `@evidencelens/contracts`.

## Numbering

`0001-` and up. Don't reuse numbers, even for withdrawn RFCs.

## Template

```markdown
# RFC NNNN: <title>

**Status:** Draft | Accepted | Withdrawn | Superseded by NNNN
**Owner:** @<github-handle>
**Affects:** proto / graphql / openapi / websocket / mcp / events

## Motivation
What problem does this solve? Why now?

## Proposal
Concrete schema / API changes. Include before / after diff snippets.

## Alternatives considered
What else was on the table and why this is preferred.

## Migration
Backward-compat story. Are old clients broken? How long do we support both?

## Rollout
1. ...
2. ...
```
