#!/usr/bin/env python3
"""R22 real human UAT signoff dry-run ledger (v1.72)."""
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
    if doc.get("ledger_status") != "R22_REAL_HUMAN_UAT_SIGNOFF_DRYRUN_LEDGER_V172":
        raise SystemExit("ledger_status must be R22_REAL_HUMAN_UAT_SIGNOFF_DRYRUN_LEDGER_V172")
    if doc.get("real_human_signoff_obtained"):
        raise SystemExit("real_human_signoff_obtained must be false")
    pilot = DOC / doc.get("pilot_doc", "")
    if not pilot.is_file():
        raise SystemExit("missing pilot_doc")
    entries = doc.get("ledger_entries", [])
    if len(entries) != int(doc.get("expected_entry_count", 0)):
        raise SystemExit("ledger entry count mismatch")
    scenarios = json.loads((FX / doc["joint_uat_scenario_fixture"]).read_text()).get("scenarios", [])
    scenario_personas = {s["persona"] for s in scenarios}
    for entry in entries:
        if "PLACEHOLDER" not in str(entry.get("signoff_ref", "")):
            raise SystemExit("signoff_ref must stay PLACEHOLDER in dry-run")
        if entry.get("persona") not in scenario_personas:
            raise SystemExit(f"persona {entry.get('persona')} must match joint UAT scenarios")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-real-human-uat-signoff-shape-rehearsal.py"),
            str(FX / doc["signoff_shape_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / doc["filled_log_fixture"])],
        check=True,
        cwd=ROOT,
    )
    log = json.loads((FX / doc["filled_log_fixture"]).read_text())
    ticket = str(log.get("production_signoff_ticket", ""))
    if "PENDING" not in ticket and "PLACEHOLDER" not in ticket:
        raise SystemExit("filled log production_signoff_ticket must stay rehearsal pending")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: R22 UAT signoff dry-run ledger ({len(entries)} entries)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()