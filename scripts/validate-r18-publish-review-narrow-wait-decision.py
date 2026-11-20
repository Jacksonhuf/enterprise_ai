#!/usr/bin/env python3
"""Wait decision for publish-review-narrow vs parent defer (v1.62)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"
MANIFEST = ROOT / "frontend/r18-baselines/manifest.json"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("decision_status") != "R18_PUBLISH_REVIEW_NARROW_WAIT_V162":
        raise SystemExit("decision_status must be R18_PUBLISH_REVIEW_NARROW_WAIT_V162")
    if doc.get("resolution") != "WAIT_ON_PUBLISH_REVIEW_PARENT_DEFER":
        raise SystemExit("resolution must be WAIT_ON_PUBLISH_REVIEW_PARENT_DEFER")
    slug = doc["slug"]
    parent = doc["parent_defer_slug"]
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r18-headroom-defer-rollup.py"), str(FX / doc["defer_rollup_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r18-narrow-only-defer-inventory.py"), str(FX / doc["narrow_inventory_fixture"])],
        check=True,
        cwd=ROOT,
    )
    rollup = json.loads((FX / doc["defer_rollup_fixture"]).read_text())
    if parent not in {r["slug"] for r in rollup["defer_records"]}:
        raise SystemExit("parent must appear in defer rollup")
    if slug in {r["slug"] for r in rollup["defer_records"]}:
        raise SystemExit("narrow slug must not have separate defer record")
    inv = json.loads((FX / doc["narrow_inventory_fixture"]).read_text())
    if slug not in inv.get("open_narrow_without_defer_slugs", []):
        raise SystemExit("slug must be in open narrow list")
    cfg = json.loads(RATIOS.read_text())
    want = float(doc["expected_max_diff_ratio"])
    if float(cfg.get(slug, 0)) != want:
        raise SystemExit("max-diff ratio mismatch")
    manifest = json.loads(MANIFEST.read_text())
    if slug not in {s["slug"] for s in manifest["spots"]}:
        raise SystemExit("manifest missing slug")
    gap_doc = ROOT / "docs/development" / doc["r18_gap_doc"]
    if slug.replace("-", "_") not in gap_doc.read_text() and slug not in gap_doc.read_text():
        raise SystemExit("gap doc must mention publish-review-narrow")
    print(f"OK: publish-review-narrow wait on {parent}")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()