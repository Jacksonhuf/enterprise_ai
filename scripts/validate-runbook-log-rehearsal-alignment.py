#!/usr/bin/env python3
"""Align runbook first-fill rehearsal table rows with production log shape fields (v1.51)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    align = json.loads(path.read_text())
    align_status = align.get("alignment_status")
    if align_status not in (
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V151",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V152",
    ):
        raise SystemExit("alignment_status must be V151 or V152")
    table_path = FX / align["rehearsal_table_fixture"]
    log_path = FX / align["production_log_shape_fixture"]
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-first-fill-rehearsal-table.py"), str(table_path)],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(log_path)],
        check=True,
        cwd=ROOT,
    )
    table = json.loads(table_path.read_text())
    log = json.loads(log_path.read_text())
    refs = {r["checklist_ref"] for r in table.get("checklist_rows", [])}
    mapped_refs = {m["checklist_ref"] for m in align.get("row_field_map", [])}
    if refs != mapped_refs:
        raise SystemExit(f"row_field_map refs {mapped_refs} != table refs {refs}")
    for mapping in align.get("row_field_map", []):
        for field in mapping.get("log_fields", []):
            if field not in log:
                raise SystemExit(f"log shape missing field {field} for {mapping['checklist_ref']}")
        hint = mapping.get("rehearsal_value_hint")
        if hint:
            row = next(r for r in table["checklist_rows"] if r["checklist_ref"] == mapping["checklist_ref"])
            val = str(row.get("rehearsal_value", ""))
            if hint not in val and mapping["checklist_ref"] not in ("main_1",):
                raise SystemExit(f"{mapping['checklist_ref']}: rehearsal_value must contain {hint}")
    table_name = align.get("rehearsal_table_fixture", "")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V151":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.50.json":
            raise SystemExit("v1.51 alignment must use v1.50 rehearsal table")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V152":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.52.json":
            raise SystemExit("v1.52 alignment must use v1.52 rehearsal table")
    print("OK: runbook log↔rehearsal table alignment")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()