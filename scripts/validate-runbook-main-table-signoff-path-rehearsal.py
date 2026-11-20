#!/usr/bin/env python3
"""Rehearsal artifact for production §3 main table #1–#8 signoff path (v1.61)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
DOC = ROOT / "docs/development"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("path_status") != "RUNBOOK_MAIN_TABLE_SIGNOFF_PATH_V161":
        raise SystemExit("path_status must be RUNBOOK_MAIN_TABLE_SIGNOFF_PATH_V161")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    for name in (doc.get("runbook_checklist_doc"), doc.get("signoff_sequence_doc")):
        if not (DOC / name).is_file():
            raise SystemExit(f"missing doc {name}")
    rows = doc.get("main_table_rows", [])
    if len(rows) != 8:
        raise SystemExit("main_table_rows must list 8 rows")
    nums = [r.get("row") for r in rows]
    if nums != [str(i) for i in range(1, 9)]:
        raise SystemExit("rows must be 1..8 in order")
    row8 = next(r for r in rows if r.get("row") == "8")
    if not row8.get("rehearsal_only") or "v1.60" not in row8.get("evidence_fixture", ""):
        raise SystemExit("row 8 must reference v1.60 main_table8 check")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-main-table8-dashboard-window-check.py"),
            str(FX / row8["evidence_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    for r in rows:
        if r.get("row") != "8" and r.get("rehearsal_checked"):
            raise SystemExit(f"row {r['row']} must remain unchecked in rehearsal path")
    log_name = doc.get("filled_log_shape_fixture", "")
    if not (FX / log_name).is_file():
        raise SystemExit("missing filled_log_shape_fixture")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / log_name)],
        check=True,
        cwd=ROOT,
    )
    print("OK: runbook main table signoff path rehearsal (#1–#8)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()