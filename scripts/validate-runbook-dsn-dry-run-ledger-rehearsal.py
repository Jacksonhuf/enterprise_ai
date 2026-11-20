#!/usr/bin/env python3
"""Rehearsal joint: DSN dry-run ledger vs filled log and gates (v1.67)."""
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
    if doc.get("rehearsal_status") != "RUNBOOK_DSN_DRY_RUN_LEDGER_V167":
        raise SystemExit("rehearsal_status must be RUNBOOK_DSN_DRY_RUN_LEDGER_V167")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-schema8-production-ledger-dsn-dry-run.py"), str(FX / doc["dsn_dry_run_ledger_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-entered-ledger-write-gate.py"), str(FX / doc["entered_write_gate_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-sre-platform-dual-signoff-shape.py"), str(FX / doc["dual_signoff_shape_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-section3-entered-warehouse-intake.py"), str(FX / doc["warehouse_intake_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run([PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / doc["filled_log_fixture"])], check=True, cwd=ROOT)
    log = json.loads((FX / doc["filled_log_fixture"]).read_text())
    dry = json.loads((FX / doc["dsn_dry_run_ledger_fixture"]).read_text())
    if doc.get("rehearsal_dsn_must_not_match_target"):
        if log.get("dsn_database_host") == dry.get("target_dsn_shape"):
            raise SystemExit("rehearsal filled log DSN must not match dry-run target")
    if "REHEARSAL" not in str(log.get("dsn_database_host", "")).upper():
        raise SystemExit("filled log must remain REHEARSAL DSN")
    if dry.get("apply_change_ticket_ref") != "CHG-ENG-WAREHOUSE-SHAPE-0001":
        raise SystemExit("dry-run ledger ticket must match warehouse shape ticket")
    print("OK: DSN dry-run ledger rehearsal joint")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()