#!/usr/bin/env python3
"""Validate implicit-default spot batch catalog policy (v1.36)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"
MANIFEST = ROOT / "frontend/r18-baselines/manifest.json"


def main() -> None:
    path = Path(sys.argv[1])
    policy = json.loads(path.read_text())
    if policy.get("catalog_status") != "IMPLICIT_DEFAULT_BATCH_POLICY_V136":
        raise SystemExit("catalog_status must be IMPLICIT_DEFAULT_BATCH_POLICY_V136")
    cfg = json.loads(RATIOS.read_text())
    default = float(cfg.get("default", 0.02))
    slugs = {s["slug"] for s in json.loads(MANIFEST.read_text())["spots"]}
    seen: set[str] = set()
    for batch in policy.get("batches", []):
        for slug in batch.get("slugs", []):
            if slug in seen:
                raise SystemExit(f"duplicate slug in batches: {slug}")
            seen.add(slug)
            if slug not in slugs:
                raise SystemExit(f"unknown slug in catalog: {slug}")
        if batch.get("manifest_version") == "v1.36":
            for slug in batch["slugs"]:
                if slug not in cfg:
                    raise SystemExit(f"v1.36 batch slug must be explicit in max-diff-ratio: {slug}")
                if float(cfg[slug]) != default:
                    raise SystemExit(f"{slug}: catalog hold must equal default {default}")
    print(f"OK: implicit-default catalog ({len(seen)} slugs across batches)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()