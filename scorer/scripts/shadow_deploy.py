"""Shadow-deploy a candidate LTR model for 24h before promotion."""
from __future__ import annotations

import argparse


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--model", required=True)
    p.add_argument("--duration-hours", type=int, default=24)
    args = p.parse_args()

    # TODO: copy model to shared storage tagged "shadow", flip
    # LTR_SHADOW_MODEL_PATH on scorer pods, monitor delta vs production
    # for 24h. If nDCG@10 doesn't regress > 1%, flip LTR_MODEL_PATH and
    # restart scorer.
    print(f"would shadow-deploy {args.model} for {args.duration_hours}h")


if __name__ == "__main__":
    main()
