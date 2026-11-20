#!/usr/bin/env python3
"""Per-item closeout rehearsal for five global matrix gaps (v1.68)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("rehearsal_status") != "R02_GLOBAL_GAP_ITEM_CLOSEOUT_V168":
        raise SystemExit("rehearsal_status must be R02_GLOBAL_GAP_ITEM_CLOSEOUT_V168")
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
        [PY, str(ROOT / "scripts/validate-r02-matrix-global-gap-r18-cross-index.py"), str(FX / doc["gap_r18_cross_index_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-figma-pixel-green-uat-rehearsal.py"), str(FX / doc["figma_uat_rehearsal_fixture"])],
        check=True,
        cwd=ROOT,
    )
    reg = json.loads((FX / doc["open_gap_registry_fixture"]).read_text())
    visual = json.loads((FX / doc["global_gap_visual_fixture"]).read_text())
    expected = set(reg.get("global_gaps_expected", []))
    panels = set(visual.get("panels", {}).get("global_blockers", []))
    if expected != panels:
        raise SystemExit("registry global gaps must match visualization blockers")
    items = doc.get("item_closeouts", [])
    if len(items) != len(expected):
        raise SystemExit("item_closeouts must cover every global gap")
    if {i["gap_id"] for i in items} != expected:
        raise SystemExit("item_closeouts gap_id set mismatch")
    for item in items:
        if item.get("closeout_rehearsal") != "PLANNED_ENGINEERING_ONLY":
            raise SystemExit("closeout must stay PLANNED_ENGINEERING_ONLY")
    if not doc.get("global_gaps_still_open"):
        raise SystemExit("global_gaps_still_open must be true")
    print(f"OK: global gap item closeout rehearsal ({len(items)} items, still open)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()