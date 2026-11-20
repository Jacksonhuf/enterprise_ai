#!/usr/bin/env python3
"""Production shape for runbook main table row #5 rollback (v1.64)."""
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
    if doc.get("shape_status") != "RUNBOOK_MAIN_TABLE_ROW5_ROLLBACK_V164":
        raise SystemExit("shape_status must be RUNBOOK_MAIN_TABLE_ROW5_ROLLBACK_V164")
    log_name = doc.get("filled_log_fixture", "")
    subprocess.run([PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / log_name)], check=True, cwd=ROOT)
    log = json.loads((FX / log_name).read_text())
    for key in doc.get("required_shape_fields", []):
        if key not in log:
            raise SystemExit(f"missing shape field {key}")
    shape = log.get("rollback_approval_shape", {})
    if not isinstance(shape, dict) or len(shape) < 3:
        raise SystemExit("rollback_approval_shape must be object with 3+ keys")
    if "PLACEHOLDER" not in str(shape.get("approved_by", "")):
        raise SystemExit("approved_by must be PLACEHOLDER in rehearsal")
    if log.get("rollback_decision_recorded") is not True:
        raise SystemExit("rollback_decision_recorded must be true for v1.64 shape")
    if not log.get("rollback_rehearsal_only"):
        raise SystemExit("rollback_rehearsal_only must be true")
    if not doc.get("production_signoff_still_placeholder"):
        raise SystemExit("production_signoff_still_placeholder must be true")
    print("OK: main table row #5 rollback production shape (rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()