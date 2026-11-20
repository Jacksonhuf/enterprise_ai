#!/usr/bin/env python3
"""Production signoff field shape template for runbook main table #1–#8 (v1.66)."""
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
    if doc.get("template_status") != "RUNBOOK_MAIN_TABLE_PRODUCTION_SIGNOFF_TEMPLATE_V166":
        raise SystemExit("template_status must be RUNBOOK_MAIN_TABLE_PRODUCTION_SIGNOFF_TEMPLATE_V166")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    if not doc.get("production_signoff_still_placeholder"):
        raise SystemExit("production_signoff_still_placeholder must be true")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-main-table-signoff-path-rehearsal.py"), str(FX / doc["signoff_path_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-main-table8-signoff-dashboard-path.py"),
            str(FX / doc["dashboard_signoff_path_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run([PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / doc["filled_log_fixture"])], check=True, cwd=ROOT)
    log = json.loads((FX / doc["filled_log_fixture"]).read_text())
    rows = doc.get("main_table_rows", [])
    if len(rows) != 8:
        raise SystemExit("main_table_rows must list 8 rows")
    for row in rows:
        if not row.get("rehearsal_production_shape_ok"):
            raise SystemExit(f"row {row['row']} must declare rehearsal_production_shape_ok")
        for field in row.get("signoff_fields", []):
            val = log.get(field)
            if val is None:
                raise SystemExit(f"row {row['row']}: missing log field {field}")
            if isinstance(val, str) and not val.strip():
                raise SystemExit(f"row {row['row']}: empty log field {field}")
    signoff = json.loads((FX / doc["signoff_path_fixture"]).read_text())
    for r in signoff.get("main_table_rows", []):
        if r.get("row") != "8" and r.get("rehearsal_checked"):
            raise SystemExit("production template must keep rows 1–7 unchecked in signoff path")
    print("OK: main table #1–#8 production signoff template (rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()