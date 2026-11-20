#!/usr/bin/env python3
"""§3 ENTERED warehouse intake production shape rehearsal (v1.65)."""
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
    if doc.get("intake_status") != "SECTION3_ENTERED_WAREHOUSE_INTAKE_V165":
        raise SystemExit("intake_status must be SECTION3_ENTERED_WAREHOUSE_INTAKE_V165")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-entered-crossref-warehouse-rehearsal.py"), str(FX / doc["entered_crossref_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-schema8-production-ledger-first-entered-rehearsal.py"), str(FX / doc["first_entered_rehearsal_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-confirmed-warehouse-bundle-rehearsal.py"), str(FX / doc["confirmed_warehouse_bundle_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-teamfill-confirmed-warehouse-window.py"), str(FX / doc["teamfill_warehouse_window_fixture"])],
        check=True,
        cwd=ROOT,
    )
    wh = json.loads((FX / doc["warehouse_crossref_fixture"]).read_text())
    if wh.get("apply_change_ticket_ref") != doc.get("apply_change_ticket_ref"):
        raise SystemExit("apply_change_ticket_ref must match warehouse crossref")
    if "REHEARSAL" in doc.get("apply_change_ticket_ref", "").upper():
        raise SystemExit("warehouse intake shape ticket must not use REHEARSAL prefix")
    first = json.loads((FX / doc["first_entered_rehearsal_fixture"]).read_text())
    if "PLACEHOLDER" not in str(first.get("verified_by_platform", "")):
        raise SystemExit("first ENTERED ledger must keep platform signoff PLACEHOLDER")
    if not doc.get("platform_verified_still_placeholder"):
        raise SystemExit("platform_verified_still_placeholder must be true")
    log_name = doc.get("filled_log_fixture", "")
    subprocess.run([PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / log_name)], check=True, cwd=ROOT)
    print(f"OK: §3 ENTERED warehouse intake shape ({doc['apply_change_ticket_ref']})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()