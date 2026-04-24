# Runbook: BYOK proxy error rate > 5%

Triggered by `BYOKProxyErrorSpike`.

## Triage

```sql
-- last 30 minutes by provider
SELECT provider, COUNT(*) AS total, SUM((error IS NOT NULL)::int) AS errors,
       AVG(duration_ms) AS mean_ms
FROM byok_proxy_telemetry
WHERE created_at > NOW() - INTERVAL '30 minutes'
GROUP BY provider;
```

## Common causes

1. **Provider-side outage** — check Anthropic / OpenAI / Groq status pages. Nothing we can do; surface a banner in the frontend if sustained.
2. **Visitor's key invalidated** — `error` rows for one provider only. Cache poisoned? Bump `AGENT_KEY_VALIDATION_CACHE_TTL_SEC` lower (default 600) so stale cached validity expires faster.
3. **Turnstile token verification failing** — the gateway rejects before reaching agent-service. Check `turnstile-verify` Worker logs. Probably bot abuse — let it ride; that's what Turnstile is for.

## Never do

- Never log keys. The telemetry table doesn't store them by design — never add a column for them.
