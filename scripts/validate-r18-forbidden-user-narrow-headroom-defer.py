#!/usr/bin/env python3
"""Validate forbidden-user-narrow v1.28 hold defer record (v1.40)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"


def main() -> None:
    path = Path(sys.argv[1])
    rec = json.loads(path.read_text())
    if rec.get("defer_status") != "FORBIDDEN_USER_NARROW_V128_HOLD":
        raise SystemExit("defer_status must be FORBIDDEN_USER_NARROW_V128_HOLD")
    slug = rec.get("slug")
    want = float(rec.get("locked_max_diff_ratio", 0))
    cfg = json.loads(RATIOS.read_text())
    if float(cfg.get(slug, 0)) != want:
        raise SystemExit(f"{slug} must remain {want} per hold")
    print(f"OK: forbidden-user-narrow defer ({slug}={want})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()