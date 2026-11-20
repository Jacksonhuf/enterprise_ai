#!/usr/bin/env python3
"""Figma pixel-green subset + UAT signoff rehearsal (v1.67)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
MANIFEST = ROOT / "frontend/r18-baselines/manifest.json"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("rehearsal_status") != "R02_FIGMA_PIXEL_GREEN_UAT_V167":
        raise SystemExit("rehearsal_status must be R02_FIGMA_PIXEL_GREEN_UAT_V167")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-figma-pending-closeout-plan.py"), str(FX / doc["figma_pending_plan_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-page-api-open-gap-registry.py"), str(FX / doc["open_gap_registry_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-matrix-global-gap-visualization.py"), str(FX / doc["global_gap_visual_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-matrix-figma-bidirectional.py"), str(FX / doc["figma_bidirectional_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-production-uat-signoff-shape.py"), str(FX / doc["uat_signoff_shape_fixture"])],
        check=True,
        cwd=ROOT,
    )
    reg = json.loads((FX / doc["open_gap_registry_fixture"]).read_text())
    remain = set(doc.get("global_gaps_remain_open", []))
    for gap in remain:
        if gap not in reg.get("global_gaps_expected", []):
            raise SystemExit(f"{gap} must stay in open gap registry")
    manifest = json.loads(MANIFEST.read_text())
    slugs = {s["slug"] for s in manifest["spots"]}
    for slug in doc.get("pixel_green_spot_slugs", []):
        if slug not in slugs:
            raise SystemExit(f"pixel_green spot missing from manifest: {slug}")
    print(f"OK: Figma pixel-green + UAT rehearsal ({len(doc.get('pixel_green_spot_slugs', []))} spots)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()