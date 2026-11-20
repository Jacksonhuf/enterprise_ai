#!/usr/bin/env python3
"""Joint shape: first ENTERED ledger and filled rehearsal apply log (v1.58)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    joint = json.loads(path.read_text())
    if joint.get("joint_status") != "ENTERED_FILLED_LOG_JOINT_V158":
        raise SystemExit("joint_status must be ENTERED_FILLED_LOG_JOINT_V158")
    if not joint.get("not_production"):
        raise SystemExit("not_production must be true")
    log_fx = joint["filled_log_fixture"]
    ledger_fx = joint["first_entered_ledger_fixture"]
    win_fx = joint["three_table_window_fixture"]
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / log_fx)],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-schema8-production-ledger-first-entered-rehearsal.py"), str(FX / ledger_fx)],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-production-gate-three-table-window.py"), str(FX / win_fx)],
        check=True,
        cwd=ROOT,
    )
    log = json.loads((FX / log_fx).read_text())
    ledger = json.loads((FX / ledger_fx).read_text())
    if log.get("change_ticket") != ledger.get("apply_change_ticket_ref"):
        raise SystemExit("log change_ticket must match ledger apply_change_ticket_ref")
    xref = json.loads(
        (FX / json.loads((FX / win_fx).read_text())["crossref_fixture"]).read_text()
    )
    ticket = xref.get("apply_change_ticket_ref", "")
    if log.get("change_ticket") != ticket:
        raise SystemExit("joint tickets must match three-table crossref")
    if joint.get("signoff_shape_rule") == "PLACEHOLDER_GATE":
        if not str(ledger.get("verified_by_platform", "")).startswith("PLACEHOLDER"):
            raise SystemExit("ledger must keep PLACEHOLDER signoff gate")
        if log.get("production_signoff_ticket") != "PENDING-SRE-SIGNOFF":
            raise SystemExit("log must keep PENDING-SRE-SIGNOFF")
    print(f"OK: ENTERED↔filled log joint ({ticket})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()