-- Scheduled queries for analytics.clicks (BigQuery free tier 1TB query/mo).
-- Materialize aggregates so the dashboard doesn't ad-hoc scan the
-- partitioned table.

-- 1. Daily click counts by variant (refreshed nightly).
CREATE OR REPLACE TABLE analytics.daily_clicks_by_variant AS
SELECT
  DATE(server_ts) AS day,
  variant,
  COUNT(*) AS clicks,
  COUNT(DISTINCT session_id) AS sessions,
  AVG(clicked_position) AS mean_position
FROM analytics.clicks
WHERE server_ts >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 90 DAY)
GROUP BY day, variant;

-- 2. Top 50 queries by click volume last 7 days.
CREATE OR REPLACE TABLE analytics.top_queries_7d AS
SELECT
  query_text,
  COUNT(*) AS clicks,
  COUNT(DISTINCT session_id) AS sessions,
  AVG(clicked_position) AS mean_position
FROM analytics.clicks
WHERE server_ts >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
GROUP BY query_text
ORDER BY clicks DESC
LIMIT 50;

-- 3. Interleaving evaluator (spec section 11.4) — paired t-test
-- helper.  Computed nightly by the eval runner; result lands in
-- analytics.interleaving_results.
SELECT
  variant,
  COUNTIF(clicked_position <= 3) / COUNT(*) AS top3_click_rate
FROM analytics.clicks
WHERE server_ts BETWEEN @start_ts AND @end_ts
GROUP BY variant;
