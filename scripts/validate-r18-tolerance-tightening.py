#!/usr/bin/env python3
"""Ensure max-diff-ratio matches tightening manifest (v1.23)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "docs/development/fixtures/r18-tolerance-tightening-v1.23.json"
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"


def main() -> None:
    plan = json.loads(MANIFEST.read_text())
    cfg = json.loads(RATIOS.read_text())
    for row in plan["tightened"]:
        slug, want = row["slug"], float(row["to"])
        if float(cfg.get(slug, 0)) != want:
            raise SystemExit(f"{slug}: expected {want}, got {cfg.get(slug)}")
    print(f"OK: R18 tightening v1.23 ({len(plan['tightened'])} spots)")
    print(f"Validated {MANIFEST}")


if __name__ == "__main__":
    main()