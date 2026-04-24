# Runbook: rollback

For each deployable, the rollback steps and the last-known-good marker.

## Cloudflare Pages (frontend)

```bash
# List deployments
wrangler pages deployment list --project-name=evidencelens-frontend

# Roll back to a prior deployment
wrangler pages deployment rollback <deployment-id> --project-name=evidencelens-frontend
```

## Cloudflare Workers (each of pubsub-bridge, click-logger, webllm-shard, turnstile-verify)

```bash
cd workers/<name>
wrangler rollback                # interactive, pick prior version
```

## GCP Cloud Run (each ingester)

```bash
# List revisions
gcloud run revisions list --service ingester-{source} --region us-central1

# Roll back: route 100% to prior revision
gcloud run services update-traffic ingester-{source} \
  --region us-central1 --to-revisions <prior-revision-name>=100
```

## NAS (Dokploy webhook deploys)

NAS services pin to image SHAs in `infra/docker-compose.nas.yml`. To roll back:

1. Find the prior SHA in the GitHub Releases page or via `docker image ls --filter reference='ghcr.io/evidencelens/*'`.
2. Update the relevant `image:` field in `infra/docker-compose.nas.yml`.
3. SSH to TrueNAS and `docker compose -f infra/docker-compose.nas.yml up -d <service>`.

## Database migrations

There are intentionally no destructive migrations. The Postgres init script (`infra/sql/001_init.sql`) is idempotent (`IF NOT EXISTS`). New columns ship as additive only. If a migration *did* break something, restore from the daily TrueNAS snapshot.
