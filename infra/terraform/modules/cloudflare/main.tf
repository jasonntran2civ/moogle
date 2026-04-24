# Cloudflare resources: R2 buckets, Pages project, Workers, KV, Tunnel.

variable "account_id" { type = string }

# ---- R2 buckets ----
resource "cloudflare_r2_bucket" "raw" {
  account_id = var.account_id
  name       = "evidencelens-raw"
  location   = "ENAM"
}

resource "cloudflare_r2_bucket" "webllm" {
  account_id = var.account_id
  name       = "evidencelens-webllm"
  location   = "ENAM"
}

# ---- Pages project ----
resource "cloudflare_pages_project" "frontend" {
  account_id        = var.account_id
  name              = "evidencelens-frontend"
  production_branch = "main"
  build_config {
    build_command   = "pnpm --filter frontend build"
    destination_dir = "frontend/.next"
    root_dir        = ""
  }
}

# ---- KV namespace (agent prompt cache) ----
resource "cloudflare_workers_kv_namespace" "cache" {
  account_id = var.account_id
  title      = "evidencelens-cache"
}

# ---- Workers (declared via wrangler.toml in workers/*/, not here) ----
# This module just creates the routes. Worker scripts deploy via
# `wrangler deploy` in the deploy-workers.yml workflow.

# ---- Tunnel ----
resource "cloudflare_tunnel" "nas" {
  account_id = var.account_id
  name       = "evidencelens-nas"
  secret     = base64sha256("evidencelens-nas-tunnel-secret-replace-me")
}

# ---- Outputs ----
output "pages_url"        { value = cloudflare_pages_project.frontend.subdomain }
output "mcp_worker_url"   { value = "https://mcp-evidencelens.${var.account_id}.workers.dev" }
output "raw_bucket"       { value = cloudflare_r2_bucket.raw.name }
output "webllm_bucket"    { value = cloudflare_r2_bucket.webllm.name }
output "kv_namespace_id"  { value = cloudflare_workers_kv_namespace.cache.id }
output "tunnel_id"        { value = cloudflare_tunnel.nas.id }
