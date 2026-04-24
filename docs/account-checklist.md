# Account Provisioning Checklist

> User-action checklist. The orchestrator (Claude) cannot perform OAuth flows, install software, or accept ToS on your behalf. This list captures everything that needs to be done by you. Update the checkboxes as you complete each item.

## Critical-path (Stream A blocker)

These gate Stream A (Platform & Infra) deploy work. Without them, code is local-only.

### GitHub

- [ ] Install GitHub CLI: <https://cli.github.com/> (Windows installer or `winget install GitHub.cli`)
- [ ] `gh auth login` — sign in with the GitHub account that will own `evidencelens/evidencelens`
- [ ] Create the public repo and push initial commit:
  ```bash
  cd c:/Trusted/evidencelens
  gh repo create evidencelens/evidencelens --public --source . --push --description "Free, public, agentic biomedical evidence search engine"
  ```
- [ ] Confirm GitHub Actions are enabled (Settings → Actions → General → Allow all actions).
- [ ] (Optional) Add me to repo collaborators if I need to push directly later.

**Alternative if no `gh` CLI:** create the repo via web at <https://github.com/new>, then locally:
```bash
cd c:/Trusted/evidencelens
git remote add origin https://github.com/<your-handle>/evidencelens.git
git push -u origin main
```
You'll need credentials (HTTPS token or SSH key).

### Cloudflare ✅ (you confirmed this is set up)

Verify these are enabled in your Cloudflare dashboard before Stream A deploy:
- [ ] R2 bucket created: `evidencelens-raw` (raw archive) and `evidencelens-webllm` (model shards)
- [ ] Pages project: `evidencelens-frontend` (will be created by `wrangler pages project create` in Stream A)
- [ ] Workers subdomain confirmed (default `<account>.workers.dev` is fine)
- [ ] Tunnel: dedicated tunnel for NAS gateway egress, named `evidencelens-nas`
- [ ] Turnstile: site key + secret key for unauthenticated LLM proxy
- [ ] KV namespace: `evidencelens-cache` (agent prompt cache)
- [ ] API token with permissions: `Workers Scripts:Edit`, `Pages:Edit`, `R2:Edit`, `KV:Edit`, `Cloudflare Tunnel:Edit`. Save to a file at `infra/secrets/cloudflare-api-token.txt` (gitignored).

## Stream A deploy-blockers (needed by ~Week 2)

### GCP (12 ingesters + Pub/Sub + BigQuery + Firestore)

- [ ] Create new GCP project: `evidencelens-prod`
- [ ] Enable APIs: Cloud Run, Pub/Sub, BigQuery, Firestore (Native mode), Cloud Build, Artifact Registry, Cloud Scheduler, IAM
- [ ] **Set billing alert at $0** — billing must be linked but no spend allowed. Cloud Console → Billing → Budgets & alerts → "Alert when actual spend > $0.01 USD" notify your email.
- [ ] Create service account: `evidencelens-deploy@evidencelens-prod.iam.gserviceaccount.com` with roles: `roles/run.admin`, `roles/pubsub.editor`, `roles/bigquery.dataEditor`, `roles/datastore.user`, `roles/iam.serviceAccountUser`, `roles/artifactregistry.writer`
- [ ] Download key JSON to `infra/secrets/gcp-deploy-sa.json` (gitignored)
- [ ] Configure Workload Identity Federation for GitHub Actions deploys (avoids long-lived keys in CI)

### Grafana Cloud

- [ ] Create free account at <https://grafana.com/auth/sign-up/create-user>
- [ ] Create stack: `evidencelens` (region nearest you — likely `prod-us-east-0`)
- [ ] Note: Loki endpoint, Tempo endpoint, Mimir endpoint, instance ID, API token (write scope)
- [ ] Save to `infra/secrets/grafana.env`

### Sentry

- [ ] Create free account at <https://sentry.io/signup/>
- [ ] Create project: `evidencelens-frontend` (platform: Next.js)
- [ ] Note: DSN
- [ ] Save to `frontend/.env.local` as `NEXT_PUBLIC_SENTRY_DSN=...`

### PostHog

- [ ] Create free account at <https://us.posthog.com/signup>
- [ ] Create project: `evidencelens`
- [ ] Note: Project API key (write), Personal API key (read for dashboard scripting)
- [ ] Save to `frontend/.env.local` as `NEXT_PUBLIC_POSTHOG_KEY=...` and `NEXT_PUBLIC_POSTHOG_HOST=https://us.i.posthog.com`

## Optional / later

### Oracle Cloud Always Free (gateway failover)

- [ ] Create Oracle Cloud account (requires credit card for ID verification, no charges if you stay in Always Free)
- [ ] Provision: 4 OCPU ARM (Ampere A1) + 24GB RAM VM in your home region
- [ ] Install Tailscale, join your tailnet
- [ ] Note: public IP for Cloudflare Load Balancer health check

### Tailscale

- [ ] Confirm tailnet is up (you've already mentioned having this)
- [ ] Generate auth key for headless devices (NAS services, Oracle VM): one tag per role (`tag:nas`, `tag:vps`, `tag:oracle`)

### NCBI / openFDA / etc. API keys (improves rate limits)

- [ ] NCBI E-utilities API key: <https://www.ncbi.nlm.nih.gov/account/settings/> (free, no rate-limit difference but tracked)
- [ ] openFDA API key: <https://open.fda.gov/apis/authentication/> (free, raises rate limit from 240/min to 240/min/key)
- [ ] CrossRef polite pool: just set `User-Agent: EvidenceLens/0.1 (mailto:<your-email>)` — no key needed
- [ ] OpenAlex polite pool: same idea, `User-Agent` with email
- [ ] Save all to `infra/secrets/api-keys.env`

## Domain — DECIDED: free subdomain (no purchase)

You chose to skip the custom domain. Final URL plan:
- Frontend: `evidencelens.pages.dev` (Cloudflare Pages auto-provisioned)
- API gateway: `gateway-evidencelens.<account>.workers.dev` (or NAS tunnel hostname)
- MCP server: `mcp-evidencelens.<account>.workers.dev`
- Recurring cost: **$0/year**

## Your action

Work through the checklist top-to-bottom. As you complete each major item, ping me and I'll move the corresponding stream forward. The next piece of work that depends on a specific account is called out in each stream's "Done when" criteria in [orchestrator-plan.md](orchestrator-plan.md).
