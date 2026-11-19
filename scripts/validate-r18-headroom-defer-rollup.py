#!/usr/bin/env python3
"""Validate headroom defer rollup against per-version defer fixtures and max-diff (v1.44)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"
FIXTURES = ROOT / "docs/development/fixtures"


def main() -> None:
    path = Path(sys.argv[1])
    rollup = json.loads(path.read_text())
    if rollup.get("rollup_status") != "HEADROOM_DEFER_ROLLUP_V144":
        raise SystemExit("rollup_status must be HEADROOM_DEFER_ROLLUP_V144")
    records = rollup.get("defer_records", [])
    if len(records) != 4:
        raise SystemExit(f"expected 4 defer records, got {len(records)}")
    cfg = json.loads(RATIOS.read_text())
    seen_slug: set[str] = set()
    for rec in records:
        fixture_name = rec.get("fixture")
        if not fixture_name:
            raise SystemExit("each record needs fixture filename")
        fp = FIXTURES / fixture_name
        if not fp.is_file():
            raise SystemExit(f"missing defer fixture: {fixture_name}")
        src = json.loads(fp.read_text())
        slug = src.get("slug")
        if slug != rec.get("slug"):
            raise SystemExit(f"{fixture_name}: slug mismatch")
        if slug in seen_slug:
            raise SystemExit(f"duplicate slug in rollup: {slug}")
        seen_slug.add(slug)
        want = float(rec.get("locked_max_diff_ratio"))
        if float(src.get("locked_max_diff_ratio", 0)) != want:
            raise SystemExit(f"{fixture_name}: locked ratio mismatch")
        if float(cfg.get(slug, 0)) != want:
            raise SystemExit(f"max-diff-ratio.json {slug} must be {want}")
        if src.get("blocking_manifest") != rec.get("blocking_manifest"):
            raise SystemExit(f"{fixture_name}: blocking_manifest mismatch")
    print(f"OK: headroom defer rollup ({len(records)} records)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()