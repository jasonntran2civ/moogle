# GitHub repo settings + Actions secrets.

variable "owner" { type = string }

resource "github_repository" "evidencelens" {
  name         = "evidencelens"
  description  = "Free, public, agentic biomedical evidence search engine"
  visibility   = "public"
  has_issues   = true
  has_projects = false
  has_wiki     = false
  topics       = ["search-engine", "biomedical", "evidence-based", "open-data", "mcp", "agentic"]
  allow_merge_commit = false
  allow_squash_merge = true
  allow_rebase_merge = false
  delete_branch_on_merge = true
  vulnerability_alerts = true
}

resource "github_branch_protection" "main" {
  repository_id = github_repository.evidencelens.node_id
  pattern       = "main"
  required_status_checks {
    strict   = true
    contexts = ["lint-proto", "guardrails"]
  }
  required_pull_request_reviews {
    required_approving_review_count = 0  # solo project; orchestrator self-reviews
  }
  enforce_admins = false
  allows_force_pushes = false
  allows_deletions = false
}
