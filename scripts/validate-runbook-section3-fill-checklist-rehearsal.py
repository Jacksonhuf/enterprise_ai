#!/usr/bin/env python3
"""Validate runbook §3 fill checklist rehearsal marks (v1.59)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DOC = ROOT / "docs/development"


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    checklist_status = doc.get("checklist_status")
    if checklist_status not in (
        "RUNBOOK_SECTION3_FILL_CHECKLIST_REHEARSAL_V159",
        "RUNBOOK_SECTION3_FILL_CHECKLIST_REHEARSAL_V160",
    ):
        raise SystemExit("checklist_status must be V159 or V160")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    marks = doc.get("rehearsal_checkmarks", {})
    c6 = marks.get("C6", {})
    if not c6.get("checked"):
        raise SystemExit("C6 must be checked in rehearsal")
    if "triple-gate" not in c6.get("evidence", ""):
        raise SystemExit("C6 evidence must reference triple-gate")
    c7 = marks.get("C7", {})
    if not c7.get("checked") or "v1.58" not in c7.get("evidence_fixture", ""):
        raise SystemExit("C7 must reference defer linkage v1.58")
    checklist_md = doc.get("runbook_doc", "")
    if not (DOC / checklist_md).is_file():
        raise SystemExit("missing runbook checklist doc")
    text = (DOC / checklist_md).read_text()
    if "| C6 |" not in text:
        raise SystemExit("runbook doc must document C6")
    if "unchecked" not in str(doc.get("production_main_table_rows", "")).lower():
        raise SystemExit("production main table must remain unchecked in rehearsal")
    if checklist_status == "RUNBOOK_SECTION3_FILL_CHECKLIST_REHEARSAL_V160":
        mt8 = marks.get("main_table_8", {})
        if not mt8.get("checked") or not mt8.get("rehearsal_only"):
            raise SystemExit("V160 must mark main_table_8 rehearsal_only")
        if "v1.60" not in mt8.get("evidence_fixture", ""):
            raise SystemExit("main_table_8 must reference v1.60 evidence")
    print("OK: runbook §3 fill checklist rehearsal")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()