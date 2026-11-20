#!/usr/bin/env python3
"""Validate runbook first production fill rehearsal table links (v1.50)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def run(script: str, fixture: str) -> None:
    subprocess.run([PY, str(ROOT / "scripts" / script), str(FX / fixture)], check=True, cwd=ROOT)


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    status = doc.get("rehearsal_status")
    if status not in (
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V150",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V152",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V156",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V157",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V158",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V159",
    ):
        raise SystemExit("rehearsal_status must be V150–V159")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    links = doc.get("links", {})
    run("validate-schema8-production-ledger-first-fill-template.py", links["first_fill_ledger_template"])
    run("validate-production-gate-ticket-crossref-team-fill.py", links["crossref_fixture"])
    run("validate-runbook-section3-defer-rollup-linkage.py", links["defer_linkage_fixture"])
    refs = {r.get("checklist_ref") for r in doc.get("checklist_rows", [])}
    for need in ("C7", "ledger_pending"):
        if need not in refs:
            raise SystemExit(f"checklist_rows must include {need}")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V152":
        for need in ("C8", "log_align"):
            if need not in refs:
                raise SystemExit(f"V152 checklist_rows must include {need}")
        align = links.get("log_alignment_fixture", "")
        if not align or not (FX / align).is_file():
            raise SystemExit("V152 requires log_alignment_fixture file")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V156":
        for need in ("C8", "C9", "log_align"):
            if need not in refs:
                raise SystemExit(f"V156 checklist_rows must include {need}")
        align = links.get("log_alignment_fixture", "")
        if "v1.56" not in align:
            raise SystemExit("V156 requires log_alignment_fixture v1.56")
        if not (FX / align).is_file():
            raise SystemExit("V156 log_alignment_fixture missing")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V157":
        for need in ("C8", "C9", "C10", "C11", "C12", "log_align"):
            if need not in refs:
                raise SystemExit(f"V157 checklist_rows must include {need}")
        align = links.get("log_alignment_fixture", "")
        if "v1.57" not in align:
            raise SystemExit("V157 requires log_alignment_fixture v1.57")
        win = links.get("three_table_window_fixture", "")
        if "v1.57" not in win:
            raise SystemExit("V157 requires three_table_window_fixture v1.57")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V158":
        for need in ("C13", "C14", "C7"):
            if need not in refs:
                raise SystemExit(f"V158 checklist_rows must include {need}")
        defer = links.get("defer_linkage_fixture", "")
        if "v1.58" not in defer:
            raise SystemExit("V158 requires defer_linkage_fixture v1.58")
        joint = links.get("entered_filled_log_joint_fixture", "")
        if "v1.58" not in joint:
            raise SystemExit("V158 requires entered_filled_log_joint_fixture v1.58")
        if "v1.58" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V158 requires log_alignment_fixture v1.58")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V159":
        for need in ("C15", "C16", "C17"):
            if need not in refs:
                raise SystemExit(f"V159 checklist_rows must include {need}")
        if "v1.59" not in links.get("fill_c6_defer_linkage_fixture", ""):
            raise SystemExit("V159 requires fill_c6_defer_linkage_fixture v1.59")
        if "v1.59" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V159 requires log_alignment_fixture v1.59")
    for row in doc.get("checklist_rows", []):
        val = str(row.get("rehearsal_value", ""))
        if "PLACEHOLDER" in val.upper():
            raise SystemExit("rehearsal_value must not use PLACEHOLDER token")
    print("OK: runbook first-fill rehearsal table")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()