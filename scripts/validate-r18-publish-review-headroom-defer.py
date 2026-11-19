#!/usr/bin/env python3
"""Validate publish-review v1.39 hold defer record (v1.42)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"


def main() -> None:
    path = Path(sys.argv[1])
    rec = json.loads(path.read_text())
    if rec.get("defer_status") != "PUBLISH_REVIEW_V139_HOLD":
        raise SystemExit("defer_status must be PUBLISH_REVIEW_V139_HOLD")
    slug = rec.get("slug")
    want = float(rec.get("locked_max_diff_ratio", 0))
    cfg = json.loads(RATIOS.read_text())
    if float(cfg.get(slug, 0)) != want:
        raise SystemExit(f"{slug} must remain {want} per hold")
    print(f"OK: publish-review defer ({slug}={want})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()