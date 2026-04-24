# Grafana Cloud dashboards + alerts.
#
# Dashboards declared as JSON files committed under infra/grafana/dashboards/.
# Alerts declared as YAML files under infra/grafana/alerts/.

locals {
  dashboards = fileset("${path.module}/../../grafana/dashboards", "*.json")
  alert_rules = fileset("${path.module}/../../grafana/alerts", "*.yaml")
}

resource "grafana_dashboard" "this" {
  for_each    = local.dashboards
  config_json = file("${path.module}/../../grafana/dashboards/${each.key}")
  overwrite   = true
}

output "dashboard_uids" {
  value = [for d in grafana_dashboard.this : d.uid]
}
