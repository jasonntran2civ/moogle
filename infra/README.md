# Infrastructure

Two compose files, two environments:

- [`docker-compose.yml`](docker-compose.yml) — local developer stack. `make dev` brings it up.
- [`docker-compose.nas.yml`](docker-compose.nas.yml) — TrueNAS production stack. Volumes mounted to `/mnt/tank/evidencelens/`.

Plus:

- [`sql/`](sql/) — Postgres init scripts (idempotent, run by `docker-entrypoint-initdb.d`).
- [`otel/`](otel/) — OpenTelemetry Collector configs (prod → Grafana Cloud, dev → stdout).
- [`prometheus/`](prometheus/) — local scrape config.
- [`grafana/`](grafana/) — dashboards + alerts (committed JSON, deployed via Terraform).
- [`terraform/`](terraform/) — Cloudflare + GCP + Grafana Cloud + GitHub provider config.
- [`bigquery/`](bigquery/) — analytics table schemas + scheduled queries.

## Bringing it up locally

```bash
cp infra/.env.example infra/.env
# Fill in placeholders (or accept defaults)
make dev
# Wait ~30s for healthchecks, then:
docker compose -f infra/docker-compose.yml ps
```

You should see `postgres`, `nats`, `redis`, `meilisearch`, `qdrant`, `neo4j`, `otel-collector` all healthy.

## Production (TrueNAS)

```bash
ssh truenas
cd /mnt/tank/evidencelens
docker compose -f infra/docker-compose.nas.yml --env-file .env up -d
```

Adjust dataset paths in the `volumes:` block if your TrueNAS layout differs.

## Secrets

Never commit `.env`. Production secrets are sops-encrypted in `infra/secrets/*.yaml.enc`. Decrypt with the team's age key (private key out-of-band; public recipient in `infra/.sops.yaml`).

## Terraform

```bash
cd infra/terraform
terraform init
terraform plan
terraform apply
```

Provider credentials read from environment per `*.tf` `variable` blocks. See [terraform/README.md](terraform/README.md) for setup.
