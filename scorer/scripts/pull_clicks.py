"""Pull last N days of clicks from BigQuery analytics.clicks for LTR training."""
from __future__ import annotations

import argparse
from google.cloud import bigquery


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--days", type=int, default=7)
    p.add_argument("--out", required=True)
    args = p.parse_args()

    client = bigquery.Client()
    sql = f"""
        SELECT session_id, query_text, clicked_doc_id, clicked_position, variant, server_ts
        FROM analytics.clicks
        WHERE server_ts >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL {args.days} DAY)
    """
    df = client.query(sql).to_dataframe()
    df.to_parquet(args.out)
    print(f"wrote {len(df)} rows -> {args.out}")


if __name__ == "__main__":
    main()
