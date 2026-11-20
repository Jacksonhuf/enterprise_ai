#!/usr/bin/env python3
"""Validate matrix global/per-entry gap visualization snapshot (v1.54)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    vis = json.loads(path.read_text())
    status = vis.get("visual_status")
    if status not in (
        "R02_GLOBAL_GAP_VISUAL_V154",
        "R02_GLOBAL_GAP_VISUAL_V156",
        "R02_GLOBAL_GAP_VISUAL_V157",
    ):
        raise SystemExit("visual_status must be V154, V156, or V157")
    matrix_path = FX / vis["matrix_fixture"]
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-page-api-acceptance-matrix.py"), str(matrix_path)],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-page-api-open-gap-registry.py"), str(FX / vis["open_gap_registry_fixture"])],
        check=True,
        cwd=ROOT,
    )
    matrix = json.loads(matrix_path.read_text())
    panels = vis.get("panels", {})
    global_list = panels.get("global_blockers", [])
    matrix_global = matrix.get("r18_global_gaps", [])
    if set(global_list) != set(matrix_global):
        raise SystemExit("global_blockers panel must match matrix r18_global_gaps")
    exp = vis.get("expected_counts", {})
    entries = matrix.get("entries", [])
    if len(entries) != int(exp.get("design_entries", 0)):
        raise SystemExit("design_entries count mismatch")
    if len(matrix_global) != int(exp.get("global_gaps", 0)):
        raise SystemExit("global_gaps count mismatch")
    with_gaps = sum(1 for e in entries if e.get("gaps"))
    if with_gaps < int(exp.get("entries_with_any_gap_min", 0)):
        raise SystemExit("entries_with_any_gap_min not met")
    closeout = panels.get("engineering_closeout_ref", "")
    if not (FX / closeout).is_file():
        raise SystemExit("engineering_closeout_ref missing")
    joint_ref = vis.get("dashboard_matrix_joint_fixture", "")
    if status in ("R02_GLOBAL_GAP_VISUAL_V156", "R02_GLOBAL_GAP_VISUAL_V157"):
        if not joint_ref or not (FX / joint_ref).is_file():
            raise SystemExit("V156+ viz requires dashboard_matrix_joint_fixture file")
    cross = vis.get("r18_gap_cross_index_fixture", "")
    if status == "R02_GLOBAL_GAP_VISUAL_V157":
        if not cross or not (FX / cross).is_file():
            raise SystemExit("V157 viz requires r18_gap_cross_index_fixture file")
    print(f"OK: global gap visualization ({len(matrix_global)} global, {with_gaps} entries w/ gaps)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()