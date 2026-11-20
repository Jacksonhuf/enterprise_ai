#!/usr/bin/env python3
"""Validate open-gap registry against matrix V150 and closeout (v1.52)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    reg = json.loads(path.read_text())
    if reg.get("registry_status") != "R02_OPEN_GAP_REGISTRY_V152":
        raise SystemExit("registry_status must be R02_OPEN_GAP_REGISTRY_V152")
    matrix_path = FX / reg["matrix_fixture"]
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-page-api-acceptance-matrix.py"), str(matrix_path)],
        check=True,
        cwd=ROOT,
    )
    matrix = json.loads(matrix_path.read_text())
    want = reg.get("matrix_status_required")
    if matrix.get("matrix_status") != want:
        raise SystemExit(f"matrix_status must be {want}")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-page-api-gap-closeout.py"), str(FX / reg["closeout_fixture"])],
        check=True,
        cwd=ROOT,
    )
    closeout = json.loads((FX / reg["closeout_fixture"]).read_text())
    if len(closeout.get("items", [])) != int(reg.get("engineering_closeout_count", 0)):
        raise SystemExit("engineering_closeout_count mismatch")
    rollup: dict[str, list[str]] = {}
    for entry in matrix.get("entries", []):
        rollup[entry["design_id"]] = list(entry.get("gaps", []))
    if not rollup:
        raise SystemExit("matrix entries required")
    global_gaps = matrix.get("r18_global_gaps", [])
    for g in reg.get("global_gaps_expected", []):
        if g not in global_gaps:
            raise SystemExit(f"missing global gap {g}")
    if matrix.get("gap_registry_link") != path.name:
        raise SystemExit("matrix gap_registry_link must point to this registry file")
    dash = reg.get("r21_dashboard_linkage_fixture", "")
    if dash and not (FX / dash).is_file():
        raise SystemExit("missing r21_dashboard_linkage_fixture file")
    viz = reg.get("global_gap_visual_fixture", "")
    if viz and not (FX / viz).is_file():
        raise SystemExit("missing global_gap_visual_fixture file")
    print(f"OK: open gap registry ({len(rollup)} entries, {len(global_gaps)} global)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()