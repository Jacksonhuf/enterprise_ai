#!/usr/bin/env python3
"""Validate services-narrow v1.34 hold defer record (v1.39)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"


def main() -> None:
    path = Path(sys.argv[1])
    rec = json.loads(path.read_text())
    if rec.get("defer_status") != "SERVICES_NARROW_V134_HOLD":
        raise SystemExit("defer_status must be SERVICES_NARROW_V134_HOLD")
    slug = rec.get("slug")
    want = float(rec.get("locked_max_diff_ratio", 0))
    cfg = json.loads(RATIOS.read_text())
    if float(cfg.get(slug, 0)) != want:
        raise SystemExit(f"{slug} must remain {want} per hold")
    print(f"OK: services-narrow defer ({slug}={want})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()