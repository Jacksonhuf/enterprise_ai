#!/usr/bin/env python3
"""Same release window: team-fill CONFIRMED + warehouse ticket format (v1.64)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    win = json.loads(path.read_text())
    if win.get("window_status") != "TEAMFILL_CONFIRMED_WAREHOUSE_WINDOW_V164":
        raise SystemExit("window_status must be TEAMFILL_CONFIRMED_WAREHOUSE_WINDOW_V164")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-production-gate-entered-confirmed-bundle-joint.py"), str(FX / win["team_fill_joint_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-production-gate-ticket-crossref-team-fill.py"), str(FX / win["team_crossref_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-confirmed-warehouse-bundle-rehearsal.py"), str(FX / win["warehouse_confirmed_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-production-gate-three-table-window.py"), str(FX / win["three_table_window_fixture"])],
        check=True,
        cwd=ROOT,
    )
    team_x = json.loads((FX / win["team_crossref_fixture"]).read_text())
    wh_x = json.loads((FX / win["warehouse_crossref_fixture"]).read_text())
    log = json.loads((FX / win["filled_log_fixture"]).read_text())
    team_change = team_x["apply_change_ticket_ref"]
    wh_change = wh_x["apply_change_ticket_ref"]
    if log.get("change_ticket") != team_change:
        raise SystemExit("filled log must match team-fill change ticket")
    if team_change == wh_change:
        raise SystemExit("team-fill and warehouse must use distinct change ticket refs")
    if "REHEARSAL" not in team_change.upper():
        raise SystemExit("team track must use REHEARSAL ticket")
    if "REHEARSAL" in wh_change.upper():
        raise SystemExit("warehouse track must use non-REHEARSAL shape ticket")
    tbl = json.loads((FX / win["three_table_window_fixture"]).read_text())
    if tbl.get("release_window_id") != win.get("release_window_id"):
        raise SystemExit("three_table_window release_window_id mismatch")
    print(f"OK: team-fill CONFIRMED + warehouse window ({team_change} / {wh_change})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()