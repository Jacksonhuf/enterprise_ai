#!/usr/bin/env python3
"""Report implicit-default R18 spots vs batch catalog policy (v1.36)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"
MANIFEST = ROOT / "frontend/r18-baselines/manifest.json"
POLICY = ROOT / "docs/development/fixtures/r18-implicit-default-spot-catalog-v1.36.json"


def main() -> None:
    out = (
        Path(sys.argv[1])
        if len(sys.argv) > 1
        else Path("/tmp/r18-implicit-default-spot-catalog.json")
    )
    cfg = json.loads(RATIOS.read_text())
    default = float(cfg.get("default", 0.02))
    slugs = [s["slug"] for s in json.loads(MANIFEST.read_text())["spots"]]
    policy = json.loads(POLICY.read_text())
    implicit = [s for s in slugs if s not in cfg]
    explicit_default = [
        s for s in slugs if s in cfg and float(cfg[s]) == default and s != "default"
    ]
    batches = []
    for batch in policy.get("batches", []):
        batches.append(
            {
                "batch_id": batch["batch_id"],
                "manifest_version": batch.get("manifest_version"),
                "slug_count": len(batch.get("slugs", [])),
                "slugs": batch.get("slugs", []),
            }
        )
    payload = {
        "catalog_report_status": "IMPLICIT_DEFAULT_CATALOG_V136",
        "default_max_diff_ratio": default,
        "spot_count": len(slugs),
        "implicit_default_count": len(implicit),
        "explicit_at_default_count": len(explicit_default),
        "implicit_default_slugs": implicit,
        "batches": batches,
        "notes": "batch-1 cataloged in v1.36; batch-2 remains implicit until next manifest",
    }
    out.write_text(json.dumps(payload, indent=2) + "\n")
    print(
        f"OK: implicit-default catalog report ({len(implicit)} implicit, "
        f"{len(explicit_default)} explicit holds)"
    )
    print(f"Wrote {out}")


if __name__ == "__main__":
    main()