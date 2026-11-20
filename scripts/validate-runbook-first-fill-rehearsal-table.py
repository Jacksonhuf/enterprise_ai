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
    ):
        raise SystemExit("rehearsal_status must be V150, V152, or V156")
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
    for row in doc.get("checklist_rows", []):
        val = str(row.get("rehearsal_value", ""))
        if "PLACEHOLDER" in val.upper():
            raise SystemExit("rehearsal_value must not use PLACEHOLDER token")
    print("OK: runbook first-fill rehearsal table")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()