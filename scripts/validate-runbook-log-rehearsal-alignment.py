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
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V156",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V157",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V158",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V159",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V160",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V161",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V162",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V163",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V164",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V165",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V166",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V167",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V168",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V169",
        "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V170",
    ):
        raise SystemExit("alignment_status must be V151–V170")
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
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V156":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.56.json":
            raise SystemExit("v1.56 alignment must use v1.56 rehearsal table")
        if "rehearsal-filled-v1.56" not in align.get("production_log_shape_fixture", ""):
            raise SystemExit("v1.56 alignment must use filled rehearsal log fixture")
        if not align.get("filled_log_gate"):
            raise SystemExit("v1.56 requires filled_log_gate")
        if log.get("disclaimer") != "REHEARSAL_ONLY — 非生产库，勿写入生产台账":
            raise SystemExit("filled log must carry REHEARSAL_ONLY disclaimer")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V157":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.57.json":
            raise SystemExit("v1.57 alignment must use v1.57 rehearsal table")
        if "rehearsal-filled-v1.57" not in align.get("production_log_shape_fixture", ""):
            raise SystemExit("v1.57 alignment must use filled log v1.57")
        if not align.get("filled_log_gate"):
            raise SystemExit("v1.57 requires filled_log_gate")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V158":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.58.json":
            raise SystemExit("v1.58 alignment must use v1.58 rehearsal table")
        if "v1.58" not in str(align.get("production_log_shape_fixture", "")) and "v1.57" not in str(
            align.get("production_log_shape_fixture", "")
        ):
            raise SystemExit("v1.58 alignment must use filled log v1.57+")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V159":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.59.json":
            raise SystemExit("v1.59 alignment must use v1.59 rehearsal table")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V160":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.60.json":
            raise SystemExit("v1.60 alignment must use v1.60 rehearsal table")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V161":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.61.json":
            raise SystemExit("v1.61 alignment must use v1.61 rehearsal table")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V162":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.62.json":
            raise SystemExit("v1.62 alignment must use v1.62 rehearsal table")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V163":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.63.json":
            raise SystemExit("v1.63 alignment must use v1.63 rehearsal table")
        if "v1.63" not in align.get("production_log_shape_fixture", ""):
            raise SystemExit("v1.63 alignment must use filled log v1.63")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V164":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.64.json":
            raise SystemExit("v1.64 alignment must use v1.64 rehearsal table")
        if "v1.64" not in align.get("production_log_shape_fixture", ""):
            raise SystemExit("v1.64 alignment must use filled log v1.64")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V165":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.65.json":
            raise SystemExit("v1.65 alignment must use v1.65 rehearsal table")
        if "v1.64" not in align.get("production_log_shape_fixture", ""):
            raise SystemExit("v1.65 alignment must use filled log v1.64+")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V166":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.66.json":
            raise SystemExit("v1.66 alignment must use v1.66 rehearsal table")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V167":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.67.json":
            raise SystemExit("v1.67 alignment must use v1.67 rehearsal table")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V168":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.68.json":
            raise SystemExit("v1.68 alignment must use v1.68 rehearsal table")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V169":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.69.json":
            raise SystemExit("v1.69 alignment must use v1.69 rehearsal table")
    if align_status == "RUNBOOK_LOG_REHEARSAL_ALIGNMENT_V170":
        if table_name != "schema8-runbook-first-fill-rehearsal-table-v1.70.json":
            raise SystemExit("v1.70 alignment must use v1.70 rehearsal table")
    for mapping in align.get("row_field_map", []):
        if mapping.get("require_log_nonempty"):
            for field in mapping.get("log_fields", []):
                val = log.get(field)
                if val is None or (isinstance(val, str) and not val.strip()):
                    raise SystemExit(f"{mapping['checklist_ref']}: log field {field} must be non-empty")
    print("OK: runbook log↔rehearsal table alignment")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()