#!/usr/bin/env python3
"""Extended filled log shape for runbook main table rows #4–#6 (v1.63)."""
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
    shape_status = doc.get("shape_status")
    if shape_status not in (
        "RUNBOOK_MAIN_TABLE_ROWS4_6_LOG_V163",
        "RUNBOOK_MAIN_TABLE_ROWS4_6_LOG_V164",
    ):
        raise SystemExit("shape_status must be V163 or V164")
    log_name = doc.get("filled_log_fixture", "")
    subprocess.run([PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / log_name)], check=True, cwd=ROOT)
    log = json.loads((FX / log_name).read_text())
    for row in doc.get("row_shape_fields", []):
        for field in row.get("log_fields", []):
            if field not in log:
                raise SystemExit(f"row {row['row']}: missing log field {field}")
        if not row.get("rehearsal_shape_ok"):
            raise SystemExit(f"row {row['row']} must declare rehearsal_shape_ok")
    r5 = next(r for r in doc["row_shape_fields"] if r["row"] == "5")
    if shape_status == "RUNBOOK_MAIN_TABLE_ROWS4_6_LOG_V163":
        if log.get("rollback_decision_recorded") is not False:
            raise SystemExit("row 5 rehearsal must keep rollback_decision_recorded false")
    else:
        r5fx = doc.get("row5_rollback_shape_fixture", "")
        if "v1.64" not in r5fx:
            raise SystemExit("V164 requires row5_rollback_shape_fixture v1.64")
        subprocess.run(
            [PY, str(ROOT / "scripts/validate-runbook-main-table-row5-rollback-shape.py"), str(FX / r5fx)],
            check=True,
            cwd=ROOT,
        )
    if "PLACEHOLDER" not in str(log.get("rollback_approver", "")):
        raise SystemExit("row 5 approver must be PLACEHOLDER in rehearsal")
    if log.get("runbook_smoke_mixed_read_ok") is not True:
        raise SystemExit("row 4 requires runbook_smoke_mixed_read_ok true")
    if log.get("r21_slo_signoff_status") != log.get("production_signoff_ticket"):
        raise SystemExit("row 6 slo status should mirror production_signoff_ticket in rehearsal")
    print("OK: main table rows #4–#6 log shape")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()