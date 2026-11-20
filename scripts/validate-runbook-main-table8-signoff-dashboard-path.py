#!/usr/bin/env python3
"""Main table row #8 signoff path vs R21 open-gap dashboard (v1.65)."""
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
    if doc.get("path_status") != "RUNBOOK_MAIN_TABLE8_SIGNOFF_DASHBOARD_PATH_V165":
        raise SystemExit("path_status must be RUNBOOK_MAIN_TABLE8_SIGNOFF_DASHBOARD_PATH_V165")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-main-table8-dashboard-window-check.py"), str(FX / doc["main_table8_check_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-main-table-signoff-path-rehearsal.py"), str(FX / doc["signoff_path_fixture"])],
        check=True,
        cwd=ROOT,
    )
    dash = json.loads((FX / doc["dashboard_linkage_fixture"]).read_text())
    if dash.get("triple_gate_window_registry_fixture") != doc.get("triple_gate_registry_fixture"):
        raise SystemExit("dashboard triple_gate registry must match path fixture")
    if dash.get("main_table_signoff_path_fixture") != doc.get("signoff_path_fixture"):
        raise SystemExit("dashboard must reference same signoff path")
    log = json.loads((FX / doc["filled_log_fixture"]).read_text())
    for key in doc.get("required_log_fields", []):
        if key not in log:
            raise SystemExit(f"missing log field {key}")
    ticket = str(log.get("production_signoff_ticket", ""))
    if "PLACEHOLDER" not in ticket and "PENDING" not in ticket:
        raise SystemExit("production_signoff_ticket must remain rehearsal placeholder")
    if not doc.get("production_signoff_still_placeholder"):
        raise SystemExit("production_signoff_still_placeholder must be true")
    print("OK: main table #8 signoff↔dashboard path rehearsal")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()