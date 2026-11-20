#!/usr/bin/env python3
"""Rehearsal bindings for main table rows #1–#7 vs filled log fields (v1.62)."""
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
    if doc.get("binding_status") != "RUNBOOK_MAIN_TABLE_ROWS1_7_LOG_V162":
        raise SystemExit("binding_status must be RUNBOOK_MAIN_TABLE_ROWS1_7_LOG_V162")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    if not doc.get("production_main_table_unchecked"):
        raise SystemExit("production_main_table_unchecked must stay true")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-main-table-signoff-path-rehearsal.py"),
            str(FX / doc["signoff_path_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-log-rehearsal-alignment.py"), str(FX / doc["log_alignment_fixture"])],
        check=True,
        cwd=ROOT,
    )
    align = json.loads((FX / doc["log_alignment_fixture"]).read_text())
    log = json.loads((FX / doc["filled_log_fixture"]).read_text())
    by_ref = {m["checklist_ref"]: m for m in align.get("row_field_map", [])}
    verified_rows = []
    for row in doc.get("row_bindings", []):
        if row.get("rehearsal_log_verified"):
            ref = row.get("alignment_ref")
            if not ref:
                raise SystemExit(f"row {row['row']} verified but no alignment_ref")
            mapping = by_ref.get(ref)
            if not mapping:
                raise SystemExit(f"missing alignment for {ref}")
            for field in mapping.get("log_fields", []):
                val = log.get(field)
                if val is None or (isinstance(val, str) and not val.strip()):
                    raise SystemExit(f"row {row['row']}: log missing {field}")
            verified_rows.append(row["row"])
        elif row.get("rehearsal_log_verified") is not False:
            raise SystemExit(f"row {row['row']} must declare rehearsal_log_verified")
    if len(verified_rows) < 3:
        raise SystemExit("expected at least rows 1–3 verified")
    signoff = json.loads((FX / doc["signoff_path_fixture"]).read_text())
    for r in signoff.get("main_table_rows", []):
        if r.get("row") != "8" and r.get("rehearsal_checked"):
            raise SystemExit("signoff path rows 1–7 must remain unchecked for production")
    print(f"OK: main table rows1–7 log rehearsal ({len(verified_rows)} rows verified)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()