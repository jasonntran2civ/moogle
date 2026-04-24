# EvidenceLens Terraform root module.
#
# Free-tier infra only. Cost ceiling $0/year — no domain purchase, no
# paid services. State stored locally + sops-encrypted backup to R2.

terraform {
  required_version = ">= 1.9"
  required_providers {
    cloudflare = { source = "cloudflare/cloudflare", version = "~> 4.48" }
    google     = { source = "hashicorp/google",     version = "~> 6.10" }
    grafana    = { source = "grafana/grafana",      version = "~> 3.13" }
    github     = { source = "integrations/github",  version = "~> 6.4" }
  }
  backend "local" {
    path = "terraform.tfstate"
  }
}

# ---- Variable definitions ----

variable "cloudflare_account_id"  { type = string,  description = "Cloudflare account id" }
variable "cloudflare_api_token"   { type = string,  description = "Cloudflare API token", sensitive = true }
variable "gcp_project"            { type = string,  description = "GCP project id" }
variable "gcp_region"             { type = string,  description = "GCP region", default = "us-central1" }
variable "grafana_cloud_url"      { type = string,  description = "Grafana Cloud stack URL" }
variable "grafana_cloud_api_key"  { type = string,  description = "Grafana Cloud API key", sensitive = true }
variable "github_owner"           { type = string,  description = "GitHub org/user owning the repo" }
variable "github_token"           { type = string,  description = "GitHub PAT", sensitive = true }

# ---- Provider config ----

provider "cloudflare" { api_token = var.cloudflare_api_token }
provider "google"     { project   = var.gcp_project, region = var.gcp_region }
provider "grafana"    { url       = var.grafana_cloud_url, auth = var.grafana_cloud_api_key }
provider "github"     { owner     = var.github_owner, token = var.github_token }

# ---- Modules ----

module "cloudflare" {
  source     = "./modules/cloudflare"
  account_id = var.cloudflare_account_id
}

module "gcp" {
  source  = "./modules/gcp"
  project = var.gcp_project
  region  = var.gcp_region
}

module "grafana" {
  source = "./modules/grafana"
}

module "github" {
  source = "./modules/github"
  owner  = var.github_owner
}

# ---- Outputs ----

output "frontend_url"      { value = module.cloudflare.pages_url }
output "mcp_url"           { value = module.cloudflare.mcp_worker_url }
output "raw_bucket"        { value = module.cloudflare.raw_bucket }
output "pubsub_topics"     { value = module.gcp.pubsub_topics }
output "cloud_run_urls"    { value = module.gcp.cloud_run_urls }
output "grafana_dashboards" { value = module.grafana.dashboard_uids }
