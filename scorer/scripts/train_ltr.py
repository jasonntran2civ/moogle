"""Train XGBoost LambdaMART on click data; emit ltr_v{date}.json."""
from __future__ import annotations

import argparse
import sys

import pandas as pd
import xgboost as xgb


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--in", dest="inp", required=True)
    p.add_argument("--out", required=True)
    args = p.parse_args()

    df = pd.read_parquet(args.inp)
    if df.empty:
        print("no clicks; skipping train", file=sys.stderr)
        sys.exit(0)

    # TODO: feature extraction matching scorer/ltr.py featurize().
    # Stub model for the pipeline.
    dmat = xgb.DMatrix(df[["clicked_position"]].values, label=df["clicked_position"].values)
    booster = xgb.train(
        {"objective": "rank:pairwise", "eta": 0.1, "max_depth": 6},
        dmat,
        num_boost_round=10,
    )
    booster.save_model(args.out)
    print(f"saved {args.out}")


if __name__ == "__main__":
    main()
