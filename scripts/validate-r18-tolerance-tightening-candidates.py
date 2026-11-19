#!/usr/bin/env python3
"""Validate tightening candidate inventory fixture (v1.38)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"
MANIFEST = ROOT / "frontend/r18-baselines/manifest.json"


def main() -> None:
    path = Path(sys.argv[1])
    policy = json.loads(path.read_text())
    if policy.get("inventory_status") != "TIGHTENING_CANDIDATE_INVENTORY_V138":
        raise SystemExit("inventory_status must be TIGHTENING_CANDIDATE_INVENTORY_V138")
    cfg = json.loads(RATIOS.read_text())
    default = float(cfg.get("default", 0.02))
    slugs = {s["slug"] for s in json.loads(MANIFEST.read_text())["spots"]}
    for slug in policy.get("exclude_from_automatic_tighten", []):
        if slug not in slugs:
            raise SystemExit(f"unknown exclude slug: {slug}")
    for slug in policy.get("pinned_priority_slugs", []):
        if slug not in slugs:
            raise SystemExit(f"unknown priority slug: {slug}")
        max_r = float(cfg.get(slug, default))
        if max_r <= default and not slug.endswith("-narrow"):
            raise SystemExit(f"priority slug must be narrow or elevated: {slug}")
    print(f"OK: tightening candidate inventory ({len(policy.get('pinned_priority_slugs', []))} pinned)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()