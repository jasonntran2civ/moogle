# Terraform

Free-tier infra orchestration for EvidenceLens.

## Setup

```bash
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars
# Fill in account ids and tokens
terraform init
terraform plan
terraform apply
```

## Variables (terraform.tfvars)

```hcl
cloudflare_account_id = "abc123..."
cloudflare_api_token  = "..."
gcp_project           = "evidencelens-prod"
gcp_region            = "us-central1"
grafana_cloud_url     = "https://stack-12345.grafana.net"
grafana_cloud_api_key = "..."
github_owner          = "your-github-handle"
github_token          = "ghp_..."
```

## Modules

- `modules/cloudflare/` — R2 buckets, Pages project, Workers KV, Tunnel.
- `modules/gcp/` — Pub/Sub topics, 12 Cloud Run ingester services, Cloud Scheduler crons, BigQuery analytics dataset, Firestore Native database.
- `modules/grafana/` — Dashboards from `infra/grafana/dashboards/*.json`.
- `modules/github/` — Repo + branch protection.

## State

Local backend for now (`terraform.tfstate`). Encrypt and back up to R2 with sops:

```bash
sops -e -i terraform.tfstate
aws s3 cp terraform.tfstate s3://evidencelens-tfstate/terraform.tfstate \
  --endpoint-url https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com
```

## Cost

Every resource is free-tier. Apply will fail loudly if any module would incur cost (e.g. Cloud Run min_instance_count > 0, BigQuery storage > 10GB).
