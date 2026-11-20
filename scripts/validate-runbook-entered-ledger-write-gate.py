#!/usr/bin/env python3
"""Rehearsal gate blocking ENTERED ledger writes without platform signoff (v1.66)."""
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
    if doc.get("gate_status") != "RUNBOOK_ENTERED_LEDGER_WRITE_GATE_V166":
        raise SystemExit("gate_status must be RUNBOOK_ENTERED_LEDGER_WRITE_GATE_V166")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-section3-entered-warehouse-intake.py"), str(FX / doc["section3_intake_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-schema8-production-ledger-first-entered-rehearsal.py"), str(FX / doc["first_entered_ledger_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run([PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / doc["filled_log_fixture"])], check=True, cwd=ROOT)
    ledger = json.loads((FX / doc["first_entered_ledger_fixture"]).read_text())
    field = doc.get("required_platform_field", "verified_by_platform")
    prefix = doc.get("required_platform_prefix", "PLACEHOLDER")
    val = str(ledger.get(field, ""))
    if not val.startswith(prefix):
        raise SystemExit(f"{field} must use {prefix} gate for rehearsal write block")
    if doc.get("write_allowed_without_platform_signoff"):
        raise SystemExit("write_allowed_without_platform_signoff must be false")
    wh = json.loads((FX / doc["warehouse_ticket_fixture"]).read_text())
    if "REHEARSAL" in wh.get("apply_change_ticket_ref", "").upper():
        raise SystemExit("warehouse ticket shape must be non-REHEARSAL for write gate")
    print("OK: ENTERED ledger write gate (platform signoff required)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()