#!/usr/bin/env python3
"""Explicit rehearsal checkmark for runbook main table row #8 vs triple-gate window (v1.60)."""
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
    check_status = doc.get("check_status")
    if check_status not in (
        "RUNBOOK_MAIN_TABLE8_DASHBOARD_WINDOW_V160",
        "RUNBOOK_MAIN_TABLE8_DASHBOARD_WINDOW_V165",
    ):
        raise SystemExit("check_status must be V160 or V165")
    if doc.get("main_table_row") != "8":
        raise SystemExit("main_table_row must be 8")
    if not doc.get("rehearsal_checked"):
        raise SystemExit("rehearsal_checked must be true")
    if doc.get("production_checked"):
        raise SystemExit("production_checked must be false in rehearsal")
    reg_name = doc.get("triple_gate_window_registry_fixture", "")
    if check_status == "RUNBOOK_MAIN_TABLE8_DASHBOARD_WINDOW_V160":
        if "v1.60" not in reg_name:
            raise SystemExit("registry fixture must be v1.60")
    else:
        if "v1.65" not in reg_name:
            raise SystemExit("V165 registry fixture must be v1.65")
        dash_name = doc.get("dashboard_linkage_fixture", "")
        if "v1.65" not in dash_name:
            raise SystemExit("V165 requires dashboard_linkage v1.65")
        if not doc.get("production_signoff_still_placeholder"):
            raise SystemExit("production_signoff_still_placeholder must be true")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r21-dashboard-triple-gate-window-registry.py"), str(FX / reg_name)],
        check=True,
        cwd=ROOT,
    )
    reg = json.loads((FX / reg_name).read_text())
    if reg.get("main_table8_check_fixture") != path.name:
        raise SystemExit("registry must point back to this main_table8 check")
    if doc.get("release_window_id") != reg.get("release_window_id"):
        raise SystemExit("release_window_id must match registry")
    checklist_md = ROOT / "docs/development" / "mysql-schema8-runbook-production-fill-checklist.md"
    if "| 8 |" not in checklist_md.read_text() and "主表" not in checklist_md.read_text():
        raise SystemExit("runbook checklist doc must describe main table rows")
    print("OK: runbook main table #8 dashboard window rehearsal check")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()