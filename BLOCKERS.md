# Resolving the three blockers

You'll see the same three lines at the bottom of every status update from this orchestration:

1. GitHub push is blocked.
2. GCP / Grafana / Sentry / PostHog accounts aren't provisioned.
3. Local language toolchains (`go`, `node`, `pnpm`, `python`/`uv`, `buf`) aren't installed in this Claude Code environment.

This document walks through resolving each one, in order, with copy-pasteable commands and the gotchas you'll hit.

---

## Block 1 — Push the repo to GitHub

**Symptom.** All commits live locally on `main`. CI hasn't run, branch protection isn't configured, deploy workflows can't fire.

**Root cause.** This Claude Code session has no `gh` CLI, no SSH key registered with GitHub, and no `credential.helper` configured for HTTPS.

### 1.1 — Easiest path: install `gh` and let it handle auth

**On Windows (winget):**

```powershell
winget install --id GitHub.cli
```

Then in your bash terminal (Git Bash / WSL / VS Code terminal):

```bash
gh auth login
# Answer the prompts:
#   GitHub.com
#   HTTPS
#   Yes, authenticate Git with your GitHub credentials
#   Login with a web browser   <-- copy the one-time code, hit Enter,
#                                   paste it in the browser window that opens
```

`gh` will install a `credential.helper` for you, so subsequent `git push` calls just work.

### 1.2 — Verify auth

```bash
gh auth status
```

Expected output:
```
github.com
  ✓ Logged in to github.com account <your-handle> (keyring)
  - Active account: true
  - Git operations protocol: https
  - Token: gho_***************************
  - Token scopes: 'gist', 'read:org', 'repo', 'workflow'
```

If `Token scopes` doesn't include `workflow`, you can't deploy:
```bash
gh auth refresh -h github.com -s workflow
```

### 1.3 — Create the public repo and push

From the repo root:

```bash
cd c:/Trusted/evidencelens

# Sanity check - you should see 11 commits and the contracts-v1.0.0 tag
git log --oneline | head -15
git tag -l

# Create the repo and push everything in one shot
gh repo create evidencelens/evidencelens \
  --public \
  --source . \
  --description "Free, public, agentic biomedical evidence search engine" \
  --homepage "https://evidencelens.pages.dev" \
  --push
```

If `evidencelens/evidencelens` is taken (org name unavailable), use your own handle:

```bash
gh repo create <your-github-handle>/evidencelens --public --source . --push
```

Then update every reference in the repo from `evidencelens/evidencelens` to your slug. Files that mention the slug:
- [`README.md`](README.md)
- [`BUILT-WITH.md`](BUILT-WITH.md)
- [`docs/launch/show-hn.md`](docs/launch/show-hn.md)
- [`docs/launch/blog-post.md`](docs/launch/blog-post.md)
- [`docs/launch/gate.md`](docs/launch/gate.md)
- [`infra/terraform/modules/github/main.tf`](infra/terraform/modules/github/main.tf)
- The `image:` paths in every service `docker-compose.yml` (`ghcr.io/evidencelens/*` → `ghcr.io/<your-handle>/*`)

A one-liner sed-fix:
```bash
git grep -l 'evidencelens/evidencelens' \
  | xargs sed -i 's#evidencelens/evidencelens#<your-handle>/evidencelens#g'
git grep -l 'ghcr.io/evidencelens/' \
  | xargs sed -i 's#ghcr.io/evidencelens/#ghcr.io/<your-handle>/#g'
git commit -am "rename: org slug -> <your-handle>"
git push
```

### 1.4 — Push the contracts-v1.0.0 tag

`gh repo create --push` only pushes the current branch. You need the tag separately:

```bash
git push origin contracts-v1.0.0
```

### 1.5 — Confirm CI runs and is green

```bash
gh run list --limit 5
gh run watch
```

You're looking for the `CI` workflow with `lint-proto` and `guardrails` jobs to go green. If `lint-proto` fails, that's the first time `buf` has actually validated the proto files end-to-end — fix forward, push, re-tag if needed (the tag is local until `git push origin contracts-v1.0.0`).

### 1.6 — Configure branch protection

The Terraform `github` module covers this, but for the immediate-term:

```bash
gh api -X PUT repos/<your-handle>/evidencelens/branches/main/protection \
  --input - <<'EOF'
{
  "required_status_checks": { "strict": true, "contexts": ["lint-proto", "guardrails"] },
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false
}
EOF
```

### 1.7 — Fallback path (no `gh` CLI)

If you can't install `gh`:

1. Create the repo via the web at <https://github.com/new>. Name it `evidencelens`, set Public, do NOT initialize with README/.gitignore/license.
2. Generate a Personal Access Token (classic) at <https://github.com/settings/tokens/new> with scopes: `repo`, `workflow`. Copy it.
3. Add the remote and push using the token as your password:
   ```bash
   cd c:/Trusted/evidencelens
   git remote add origin https://github.com/<your-handle>/evidencelens.git
   git push -u origin main
   # Username: <your-handle>
   # Password: <paste the PAT, not your GitHub password>
   git push origin contracts-v1.0.0
   ```
4. Cache the credentials so subsequent pushes don't re-prompt:
   ```bash
   git config --global credential.helper "store"   # plaintext - quick + dirty
   # OR (Windows; preferred)
   git config --global credential.helper manager-core
   ```

### 1.8 — Alternative: SSH-based push

If you'd rather avoid PATs entirely:

```bash
# Generate a key
ssh-keygen -t ed25519 -C "evidencelens-deploy" -f ~/.ssh/id_ed25519_github
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519_github

# Add to GitHub
gh ssh-key add ~/.ssh/id_ed25519_github.pub --title "evidencelens-deploy"
# (Or paste it via the web at https://github.com/settings/ssh/new)

# Configure git to use SSH for github.com
cat >> ~/.ssh/config <<'EOF'
Host github.com
  HostName github.com
  User git
  IdentityFile ~/.ssh/id_ed25519_github
  IdentitiesOnly yes
EOF

# Push with SSH remote
git remote set-url origin git@github.com:<your-handle>/evidencelens.git
git push -u origin main
git push origin contracts-v1.0.0
```

---

## Block 2 — Provision the free-tier accounts

**Symptom.** Stream A deploy workflows fail because secrets aren't set; observability has nowhere to send traces; the LLM proxy can't validate Turnstile tokens.

**Root cause.** Each free-tier service requires you (a human) to OAuth in, accept ToS, and save credentials. Claude Code can't do those steps for you.

The complete inventory is in [`docs/account-checklist.md`](docs/account-checklist.md). This section gives you a sequenced 60-minute walkthrough plus the exact `gh secret set` commands to wire each one to GitHub Actions.

> **One-time prep.** Set an env var so the rest of the commands are copy-pasteable:
> ```bash
> export GH_REPO=<your-handle>/evidencelens
> ```

### 2.1 — Cloudflare (5 min, you said this is already set up)

Confirm and capture credentials:

1. Find your **Account ID**: <https://dash.cloudflare.com/> → any zone → right-side panel → "Account ID".
2. Create an **API token** at <https://dash.cloudflare.com/profile/api-tokens> → "Create Token" → "Edit Cloudflare Workers" template, add R2 + KV + Pages permissions:
   - Account → Workers Scripts → Edit
   - Account → Cloudflare Pages → Edit
   - Account → Workers R2 Storage → Edit
   - Account → Workers KV Storage → Edit
   - Account → Cloudflare Tunnel → Edit
3. Create R2 buckets (UI: <https://dash.cloudflare.com/?to=/:account/r2>):
   - `evidencelens-raw`
   - `evidencelens-webllm`
4. Create Turnstile widget: <https://dash.cloudflare.com/?to=/:account/turnstile> → Add site → "Always Challenge". Keep both the **Site Key** (public) and **Secret Key**.
5. Create a KV namespace: `wrangler kv:namespace create evidencelens-cache` — note the returned `id`.
6. (Optional now) Create the Tunnel: `cloudflared tunnel create evidencelens-nas`.

Push to GitHub Actions secrets:

```bash
gh secret set CF_ACCOUNT_ID -b "<your-account-id>" -R $GH_REPO
gh secret set CF_API_TOKEN -b "<your-api-token>" -R $GH_REPO

# Push the Turnstile secret to the worker via wrangler (separate from GH secrets)
cd workers/turnstile-verify && wrangler secret put TURNSTILE_SECRET
# Paste your Turnstile secret key when prompted
cd ../..

# Update the KV namespace id in workers/click-logger/wrangler.toml:
sed -i "s|REPLACE_WITH_KV_NAMESPACE_ID|<the-kv-id>|" workers/click-logger/wrangler.toml
git commit -am "infra: bind click-logger to real KV namespace"
git push
```

### 2.2 — GCP project (15 min — the trickiest)

Create the project:

```bash
# Install gcloud if you don't have it: https://cloud.google.com/sdk/docs/install
gcloud projects create evidencelens-prod --name="EvidenceLens production"
gcloud config set project evidencelens-prod
```

**Link a billing account.** GCP requires this even for free-tier — but you'll set a $0 alert below so nothing actually charges.

```bash
# List your billing accounts
gcloud billing accounts list
# Link one to the project
gcloud billing projects link evidencelens-prod --billing-account=<XXXXXX-XXXXXX-XXXXXX>
```

**Set a $0.01 budget alert** (this is the safety net — if anything starts spending, you get an email):

1. Go to <https://console.cloud.google.com/billing/budgets?project=evidencelens-prod>
2. Create budget → Period: Monthly → Budget amount: $1 → Alert at: 1% of actual ($0.01) → email yourself.

**Enable APIs:**

```bash
gcloud services enable \
  run.googleapis.com \
  pubsub.googleapis.com \
  bigquery.googleapis.com \
  firestore.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  cloudscheduler.googleapis.com \
  iam.googleapis.com \
  iamcredentials.googleapis.com \
  sts.googleapis.com
```

**Create the deploy service account:**

```bash
gcloud iam service-accounts create evidencelens-deploy \
  --display-name="EvidenceLens deploy SA"

# Grant the roles per docs/account-checklist.md
for role in roles/run.admin roles/pubsub.editor roles/bigquery.dataEditor \
            roles/datastore.user roles/iam.serviceAccountUser \
            roles/artifactregistry.writer roles/cloudscheduler.admin; do
  gcloud projects add-iam-policy-binding evidencelens-prod \
    --member="serviceAccount:evidencelens-deploy@evidencelens-prod.iam.gserviceaccount.com" \
    --role="$role"
done
```

**Set up Workload Identity Federation for GitHub Actions** (avoids long-lived keys in CI):

```bash
gcloud iam workload-identity-pools create gh-pool \
  --location="global" --display-name="GitHub Actions pool"

gcloud iam workload-identity-pools providers create-oidc gh-provider \
  --location="global" \
  --workload-identity-pool="gh-pool" \
  --display-name="GitHub OIDC" \
  --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository" \
  --issuer-uri="https://token.actions.githubusercontent.com"

# Bind the SA to your repo
PROJECT_NUMBER=$(gcloud projects describe evidencelens-prod --format="value(projectNumber)")
gcloud iam service-accounts add-iam-policy-binding \
  evidencelens-deploy@evidencelens-prod.iam.gserviceaccount.com \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/gh-pool/attribute.repository/$GH_REPO"

WIF_PROVIDER="projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/gh-pool/providers/gh-provider"
gh secret set GCP_WIF_PROVIDER -b "$WIF_PROVIDER" -R $GH_REPO
gh secret set GCP_DEPLOY_SA -b "evidencelens-deploy@evidencelens-prod.iam.gserviceaccount.com" -R $GH_REPO
gh secret set GCP_PROJECT -b "evidencelens-prod" -R $GH_REPO
```

**For local Terraform / dev, also create a key file:**

```bash
gcloud iam service-accounts keys create ~/.config/gcloud/evidencelens-sa.json \
  --iam-account=evidencelens-deploy@evidencelens-prod.iam.gserviceaccount.com
# This is the file referenced by GOOGLE_APPLICATION_CREDENTIALS in infra/.env
```

**Create the Artifact Registry repo for ingester images:**

```bash
gcloud artifacts repositories create evidencelens \
  --repository-format=docker \
  --location=us-central1 \
  --description="EvidenceLens ingester images"
```

**Initialize Firestore in Native mode:**

```bash
gcloud firestore databases create --location=nam5 --type=firestore-native
```

### 2.3 — Grafana Cloud (5 min)

1. Sign up at <https://grafana.com/auth/sign-up/create-user>.
2. Create a stack: name `evidencelens`, region nearest you (likely `prod-us-east-0`).
3. From the stack details page, copy:
   - **Loki**: Send Logs URL + user (instance ID) + API token
   - **Tempo**: Send Traces URL + user + API token
   - **Mimir**: Send Metrics URL + user + API token
   - **Grafana**: Stack URL (e.g. `https://stack-12345.grafana.net`) + a write API key (Admin → API keys → Add API key)

Save them to `infra/.env` and to GH secrets:

```bash
gh secret set GRAFANA_CLOUD_URL -b "<stack-url>" -R $GH_REPO
gh secret set GRAFANA_CLOUD_API_KEY -b "<api-key>" -R $GH_REPO
gh secret set GRAFANA_LOKI_URL -b "<loki-url>" -R $GH_REPO
gh secret set GRAFANA_LOKI_USER -b "<loki-user>" -R $GH_REPO
gh secret set GRAFANA_LOKI_PASSWORD -b "<loki-token>" -R $GH_REPO
gh secret set GRAFANA_TEMPO_URL -b "<tempo-url>" -R $GH_REPO
gh secret set GRAFANA_TEMPO_USER -b "<tempo-user>" -R $GH_REPO
gh secret set GRAFANA_TEMPO_PASSWORD -b "<tempo-token>" -R $GH_REPO
gh secret set GRAFANA_MIMIR_URL -b "<mimir-url>" -R $GH_REPO
gh secret set GRAFANA_MIMIR_USER -b "<mimir-user>" -R $GH_REPO
gh secret set GRAFANA_MIMIR_PASSWORD -b "<mimir-token>" -R $GH_REPO
```

### 2.4 — Sentry (3 min)

1. Sign up at <https://sentry.io/signup/>.
2. Create project: `evidencelens-frontend`, platform Next.js.
3. Copy the DSN.
4. Save as a GitHub repo secret (the deploy-frontend workflow injects it into `NEXT_PUBLIC_SENTRY_DSN`):

```bash
gh secret set NEXT_PUBLIC_SENTRY_DSN -b "<the-dsn>" -R $GH_REPO
```

### 2.5 — PostHog (3 min)

1. Sign up at <https://us.posthog.com/signup>.
2. Create project: `evidencelens`.
3. Copy the Project API key (write).
4. Save as repo secrets:

```bash
gh secret set NEXT_PUBLIC_POSTHOG_KEY -b "<the-key>" -R $GH_REPO
gh secret set NEXT_PUBLIC_POSTHOG_HOST -b "https://us.i.posthog.com" -R $GH_REPO
```

### 2.6 — Apply Terraform (5 min)

With every variable now in your environment, populate `infra/terraform/terraform.tfvars` and apply:

```bash
cd infra/terraform
cat > terraform.tfvars <<EOF
cloudflare_account_id = "$CF_ACCOUNT_ID"
cloudflare_api_token  = "$CF_API_TOKEN"
gcp_project           = "evidencelens-prod"
gcp_region            = "us-central1"
grafana_cloud_url     = "$GRAFANA_CLOUD_URL"
grafana_cloud_api_key = "$GRAFANA_CLOUD_API_KEY"
github_owner          = "<your-handle>"
github_token          = "$GH_TOKEN"   # gh auth token --hostname github.com
EOF

terraform init
terraform plan -out=plan.tfplan
# Review the plan output. Should only show resources being created, no
# destroys. If you see anything that would cost money, abort.
terraform apply plan.tfplan
```

### 2.7 — Encrypt and back up state

```bash
# Generate an age key (one-time, save the private key OUT OF BAND)
age-keygen -o ~/.config/sops/age/keys.txt
PUB=$(grep -oP 'public key: \K.*' ~/.config/sops/age/keys.txt)
sed -i "s|age1example_replace_with_real_recipient_public_key|$PUB|g" infra/.sops.yaml

# Encrypt and back up tfstate to R2
sops -e infra/terraform/terraform.tfstate > infra/terraform/terraform.tfstate.enc
aws --endpoint-url https://$CF_ACCOUNT_ID.r2.cloudflarestorage.com \
    s3 cp infra/terraform/terraform.tfstate.enc s3://evidencelens-tfstate/terraform.tfstate.enc
```

### 2.8 — Trigger the deploys

After the next `git push`, deploy workflows fire automatically:

- `deploy-frontend.yml` deploys to Cloudflare Pages.
- `deploy-workers.yml` deploys all four Workers.
- `deploy-cloud-run.yml` deploys all 12 ingesters.
- `deploy-nas.yml` POSTs to the Dokploy webhook (you need to configure that webhook URL + token in your Dokploy dashboard, then set `DOKPLOY_WEBHOOK_URL` and `DOKPLOY_TOKEN` GH secrets).

```bash
# Force-trigger if needed
gh workflow run deploy-frontend.yml -R $GH_REPO
gh workflow run deploy-workers.yml -R $GH_REPO
gh workflow run deploy-cloud-run.yml -R $GH_REPO
```

---

## Block 3 — Install local language toolchains

**Symptom.** You can't run `make lint`, `make test`, `make smoke`, or any per-service dev command on your workstation. CI runs everything fine; your local dev loop is dead.

**Root cause.** This Claude Code session is shipped without the language toolchains installed. Your real workstation almost certainly already has most of them, but here's a clean install per OS.

### 3.1 — Windows (winget) — recommended

```powershell
# Run from PowerShell as your normal user (not Admin)
winget install --id Git.Git
winget install --id GitHub.cli
winget install --id GoLang.Go.1.23
winget install --id OpenJS.NodeJS.LTS         # Node 20+
winget install --id Python.Python.3.12
winget install --id astral-sh.uv               # Python package manager
winget install --id pnpm.pnpm                  # JS workspace manager
winget install --id Bufbuild.Buf               # Protobuf
winget install --id Hashicorp.Terraform        # Infra
winget install --id Mozilla.sops               # Secret management
winget install --id Cloudflare.cloudflared     # Tunnel
winget install --id Docker.DockerDesktop       # Local compose stack
```

After installation, **close and reopen your terminal** so PATH refreshes.

### 3.2 — macOS (brew)

```bash
brew install gh go node@20 python@3.12 uv pnpm buf terraform sops cloudflared
brew install --cask docker
brew install hashicorp/tap/terraform
```

### 3.3 — Verify

```bash
gh --version       # 2.x
go version         # go1.23.x
node --version     # v20.x or v22.x
pnpm --version     # 9.x
python --version   # 3.12.x
uv --version
buf --version
terraform --version
sops --version
docker --version
```

If any are missing, search the project's package manager (`winget search ...` or `brew search ...`) and install individually.

### 3.4 — One-time repo setup after toolchains are installed

From the repo root:

```bash
# Workspaces
pnpm install                    # installs frontend / gateway / mcp-server / workers / contracts
go work sync                    # links ingest/ + index/ Go modules

# Per-Python service
for svc in process embedder scorer agent; do
  (cd $svc && uv sync)
done

# Generate proto stubs
make proto                      # buf lint + buf generate -> proto/gen/{go,python,typescript}
make contracts                  # build the @evidencelens/contracts package

# Lint everything
make lint
```

### 3.5 — Bring up the local dev stack

```bash
cp infra/.env.example infra/.env
# Fill in placeholders (or accept defaults for purely-local services)

make dev                        # docker compose up -d for the data plane
docker compose -f infra/docker-compose.yml ps   # all healthy
```

In separate terminals:

```bash
# Gateway dev server
pnpm --filter evidencelens-gateway start:dev    # http://localhost:8080

# Frontend dev server
pnpm --filter evidencelens-frontend dev          # http://localhost:3000

# Processor
(cd process && uv run python main.py)

# Embedder (CPU fallback path; GPU is a separate journey)
(cd embedder && uv run python main.py)

# Scorer
(cd scorer && uv run python main.py)
```

Smoke test:

```bash
curl -s "http://localhost:8080/api/search?q=heart+failure&top_k=5" | jq .
# Expected: SearchResult JSON with results[] (may be empty until you've run an ingester)

# Trigger ingester-pubmed once
(cd ingest && go run ./cmd/ingester-pubmed) &
sleep 2 && curl -X POST http://localhost:8080/run

# Wait ~30s for the pipeline to flow, then re-search
curl -s "http://localhost:8080/api/search?q=cardiology" | jq '.results | length'
```

### 3.6 — IDE wiring

Open the repo root in VS Code (or your IDE of choice). The shipped `.editorconfig` + the workspace files (`pnpm-workspace.yaml`, `go.work`) make multi-language IntelliSense work without further config. Recommended VS Code extensions:

- `golang.go`
- `ms-python.python` + `ms-python.vscode-pylance`
- `dbaeumer.vscode-eslint` + `esbenp.prettier-vscode`
- `bradlc.vscode-tailwindcss`
- `bufbuild.vscode-buf`
- `hashicorp.terraform`

---

## After all three blockers are clear

Run the full launch-gate checklist at [`docs/launch/gate.md`](docs/launch/gate.md). Specifically:

```bash
# Run the eval against your live staging gateway
GATEWAY_URL=https://gateway-evidencelens.<account>.workers.dev python eval/run.py
# Expect nDCG@10 average >= 0.65 across the 10 queries.

# Run the load test
k6 run --env GATEWAY_URL=$GATEWAY_URL tests/load/search.js
# Expect first_wave_ms p95 < 250, http_req_duration{name:search} p95 < 800.

# Watch the SLO panels in Grafana for 7 consecutive days.
```

When that's all green, ship the launch posts ([`docs/launch/show-hn.md`](docs/launch/show-hn.md), [`docs/launch/blog-post.md`](docs/launch/blog-post.md)), point the world at `evidencelens.pages.dev`, and respond to the inevitable Hacker News questions about the COI matching policy.

Good luck.
