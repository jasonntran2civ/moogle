# GCP resources: Pub/Sub topics, Cloud Run jobs (per ingester), BigQuery
# dataset + tables, Firestore.

variable "project" { type = string }
variable "region"  { type = string }

locals {
  ingesters = [
    "pubmed", "preprint", "trials", "ictrp", "fda", "openalex",
    "crossref", "unpaywall", "nih-reporter", "open-payments",
    "cochrane", "guidelines",
  ]
}

# ---- API enablement ----
resource "google_project_service" "apis" {
  for_each = toset([
    "run.googleapis.com",
    "pubsub.googleapis.com",
    "bigquery.googleapis.com",
    "firestore.googleapis.com",
    "cloudbuild.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudscheduler.googleapis.com",
  ])
  project = var.project
  service = each.key
  disable_on_destroy = false
}

# ---- Pub/Sub topics ----
resource "google_pubsub_topic" "raw_docs"        { name = "raw-docs",        project = var.project }
resource "google_pubsub_topic" "click_events"    { name = "click-events",    project = var.project }
resource "google_pubsub_topic" "citation_edges"  { name = "citation-edges",  project = var.project }

# ---- Pub/Sub subscriptions (push to pubsub-bridge Worker) ----
resource "google_pubsub_subscription" "raw_docs_bridge" {
  name    = "raw-docs.bridge"
  topic   = google_pubsub_topic.raw_docs.name
  project = var.project
  ack_deadline_seconds = 60
  push_config {
    push_endpoint = "https://pubsub-bridge-evidencelens.workers.dev/pubsub/raw-docs"
    oidc_token { service_account_email = google_service_account.pubsub_pusher.email }
  }
  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }
}

resource "google_service_account" "pubsub_pusher" {
  account_id   = "pubsub-pusher"
  display_name = "Pub/Sub push to Cloudflare Worker"
  project      = var.project
}

# ---- Cloud Run services (one per ingester) ----
# Image lives at us-central1-docker.pkg.dev/$PROJECT/evidencelens/ingester-{source}:latest
resource "google_cloud_run_v2_service" "ingester" {
  for_each = toset(local.ingesters)
  name     = "ingester-${each.key}"
  location = var.region
  project  = var.project
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    containers {
      image = "${var.region}-docker.pkg.dev/${var.project}/evidencelens/ingester-${each.key}:latest"
      resources {
        limits = { cpu = "1", memory = "512Mi" }
        cpu_idle = true
      }
      env {
        name  = "PUBSUB_TOPIC_RAW_DOCS"
        value = google_pubsub_topic.raw_docs.name
      }
    }
    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }
    timeout = "900s"
  }
}

# ---- Cloud Scheduler (one per ingester, GHA also schedules backup) ----
resource "google_cloud_scheduler_job" "ingester" {
  for_each = toset(local.ingesters)
  name     = "ingester-${each.key}-cron"
  region   = var.region
  project  = var.project
  schedule = lookup({
    "pubmed"        = "0 */6 * * *"   # every 6h
    "preprint"      = "0 */12 * * *"
    "trials"        = "0 3 * * *"     # nightly
    "ictrp"         = "0 4 * * 0"     # weekly Sun
    "fda"           = "*/30 * * * *"  # every 30 min for recall priority
    "openalex"      = "0 5 * * 0"     # weekly
    "crossref"      = "0 6 * * *"
    "unpaywall"     = "0 7 * * *"
    "nih-reporter"  = "0 8 * * 0"
    "open-payments" = "0 9 1 * *"     # monthly check, annual real refresh
    "cochrane"      = "0 10 * * 0"
    "guidelines"    = "0 11 * * 0"
  }, each.key, "0 12 * * 0")
  http_target {
    uri         = google_cloud_run_v2_service.ingester[each.key].uri
    http_method = "POST"
    oidc_token { service_account_email = google_service_account.scheduler.email }
  }
}

resource "google_service_account" "scheduler" {
  account_id   = "ingester-scheduler"
  display_name = "Scheduler invoker for ingesters"
  project      = var.project
}

# ---- BigQuery dataset + clicks table ----
resource "google_bigquery_dataset" "analytics" {
  dataset_id = "analytics"
  project    = var.project
  location   = "US"
}

resource "google_bigquery_table" "clicks" {
  dataset_id = google_bigquery_dataset.analytics.dataset_id
  table_id   = "clicks"
  project    = var.project
  time_partitioning { type = "DAY", field = "server_ts" }
  clustering = ["variant", "query_text"]
  schema = file("${path.module}/clicks_schema.json")
  deletion_protection = false
}

# ---- Pub/Sub -> BigQuery direct subscription (drains click-events) ----
resource "google_pubsub_subscription" "clicks_to_bq" {
  name    = "click-events.bigquery"
  topic   = google_pubsub_topic.click_events.name
  project = var.project
  ack_deadline_seconds = 60

  bigquery_config {
    table             = "${var.project}.${google_bigquery_dataset.analytics.dataset_id}.${google_bigquery_table.clicks.table_id}"
    use_table_schema  = true
    write_metadata    = false
    drop_unknown_fields = true
  }

  depends_on = [
    google_bigquery_table.clicks,
    google_project_iam_member.pubsub_bq_writer,
  ]
}

# Pub/Sub service agent needs roles/bigquery.dataEditor + .metadataViewer
data "google_project" "this" {
  project_id = var.project
}

resource "google_project_iam_member" "pubsub_bq_writer" {
  project = var.project
  role    = "roles/bigquery.dataEditor"
  member  = "serviceAccount:service-${data.google_project.this.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "pubsub_bq_metadata_viewer" {
  project = var.project
  role    = "roles/bigquery.metadataViewer"
  member  = "serviceAccount:service-${data.google_project.this.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

# ---- Materialized aggregates: daily clicks-by-variant + top-queries-7d ----
resource "google_bigquery_table" "daily_clicks_by_variant" {
  dataset_id = google_bigquery_dataset.analytics.dataset_id
  table_id   = "daily_clicks_by_variant"
  project    = var.project
  schema = jsonencode([
    { name = "day",            type = "DATE",      mode = "REQUIRED" },
    { name = "variant",        type = "STRING",    mode = "NULLABLE" },
    { name = "clicks",         type = "INT64",     mode = "REQUIRED" },
    { name = "sessions",       type = "INT64",     mode = "REQUIRED" },
    { name = "mean_position",  type = "FLOAT64",   mode = "NULLABLE" },
  ])
  deletion_protection = false
}

resource "google_bigquery_data_transfer_config" "daily_aggregate" {
  display_name           = "EvidenceLens daily click aggregate"
  data_source_id         = "scheduled_query"
  schedule               = "every day 04:00"
  destination_dataset_id = google_bigquery_dataset.analytics.dataset_id
  project                = var.project
  location               = "US"
  params = {
    query = <<-SQL
      CREATE OR REPLACE TABLE analytics.daily_clicks_by_variant AS
      SELECT
        DATE(server_ts) AS day,
        variant,
        COUNT(*) AS clicks,
        COUNT(DISTINCT session_id) AS sessions,
        AVG(clicked_position) AS mean_position
      FROM analytics.clicks
      WHERE server_ts >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 90 DAY)
      GROUP BY day, variant
    SQL
  }
}

# ---- Firestore (Native mode, default region per project) ----
resource "google_firestore_database" "ab" {
  name        = "(default)"
  project     = var.project
  location_id = "nam5"
  type        = "FIRESTORE_NATIVE"
}

# ---- Outputs ----
output "pubsub_topics" {
  value = {
    raw_docs       = google_pubsub_topic.raw_docs.name
    click_events   = google_pubsub_topic.click_events.name
    citation_edges = google_pubsub_topic.citation_edges.name
  }
}

output "cloud_run_urls" {
  value = { for s in local.ingesters : s => google_cloud_run_v2_service.ingester[s].uri }
}
