#!/usr/bin/env python3
"""Warehouse crossref joint + team-fill ENTERED rehearsal (v1.62)."""
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
    if doc.get("rehearsal_status") != "ENTERED_CROSSREF_WAREHOUSE_V162":
        raise SystemExit("rehearsal_status must be ENTERED_CROSSREF_WAREHOUSE_V162")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-production-gate-crossref-bundle-joint.py"),
            str(FX / doc["warehouse_joint_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-production-gate-ticket-crossref-team-fill.py"), str(FX / doc["team_fill_crossref_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-schema8-production-ledger-first-entered-rehearsal.py"),
            str(FX / doc["first_entered_rehearsal_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    ledger = json.loads((FX / doc["entered_ledger_fixture"]).read_text())
    if ledger.get("ledger_status") != "ENTERED":
        raise SystemExit("entered ledger must be ENTERED")
    team_xref = json.loads((FX / doc["team_fill_crossref_fixture"]).read_text())
    team_change = team_xref["apply_change_ticket_ref"]
    log = json.loads((FX / doc["filled_log_fixture"]).read_text())
    if log.get("change_ticket") != team_change:
        raise SystemExit("filled log must match team-fill change ticket")
    joint = json.loads((FX / doc["warehouse_joint_fixture"]).read_text())
    wh_ticket = joint.get("apply_change_ticket_ref", "")
    if "REHEARSAL" in wh_ticket.upper():
        raise SystemExit("warehouse joint must use non-REHEARSAL shape tickets")
    if team_change == wh_ticket:
        raise SystemExit("warehouse and team-fill tracks must use distinct ticket refs in rehearsal")
    print(f"OK: ENTERED warehouse crossref rehearsal (team {team_change} / warehouse {wh_ticket})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()