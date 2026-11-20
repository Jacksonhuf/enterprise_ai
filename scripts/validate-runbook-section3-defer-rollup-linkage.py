#!/usr/bin/env python3
"""Validate runbook §3 checklist hooks against defer rollup and three-table window (v1.47+)."""
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
    link = json.loads(path.read_text())
    status = link.get("linkage_status")
    if status not in (
        "RUNBOOK_SECTION3_DEFER_ROLLUP_V147",
        "RUNBOOK_SECTION3_DEFER_ROLLUP_V149",
        "RUNBOOK_SECTION3_DEFER_ROLLUP_V158",
    ):
        raise SystemExit("linkage_status must be V147, V149, or V158")
    for doc in link.get("runbook_docs", []):
        if not (ROOT / "docs/development" / doc).is_file():
            raise SystemExit(f"missing runbook doc: {doc}")
    run("validate-r18-headroom-defer-rollup.py", link["defer_rollup_fixture"])
    run("validate-production-gate-three-table-window.py", link["three_table_window_fixture"])
    if status == "RUNBOOK_SECTION3_DEFER_ROLLUP_V147":
        run("validate-staging-team-filled-rehearsal.py", link["staging_team_fill_fixture"])
        window = json.loads((FX / link["three_table_window_fixture"]).read_text())
        staging = json.loads((FX / link["staging_team_fill_fixture"]).read_text())
        if window.get("release_window_id") != staging.get("release_window_id"):
            raise SystemExit("team fill window must match three-table window id")
    if status in ("RUNBOOK_SECTION3_DEFER_ROLLUP_V149", "RUNBOOK_SECTION3_DEFER_ROLLUP_V158"):
        xref_name = link.get("crossref_fixture", "")
        if not xref_name:
            raise SystemExit("crossref_fixture required for V149+")
        run("validate-production-gate-ticket-crossref-team-fill.py", xref_name)
        tmpl = link.get("first_fill_ledger_template", "")
        if not tmpl:
            raise SystemExit("first_fill_ledger_template required for V149+")
        run("validate-schema8-production-ledger-first-fill-template.py", tmpl)
    if status == "RUNBOOK_SECTION3_DEFER_ROLLUP_V149":
        if link.get("three_table_window_fixture") != "production-gate-three-table-window-v1.48.json":
            raise SystemExit("V149 linkage must use production-gate-three-table-window-v1.48.json")
    if status == "RUNBOOK_SECTION3_DEFER_ROLLUP_V158":
        win = link.get("three_table_window_fixture", "")
        if "v1.57" not in win:
            raise SystemExit("V158 defer must use three-table window v1.57")
        filled = link.get("filled_log_ticket_linkage_fixture", "")
        joint = link.get("entered_filled_log_joint_fixture", "")
        services = link.get("r18_services_wide_decision_fixture", "")
        if not filled or not joint or not services:
            raise SystemExit("V158 requires filled log, entered/log joint, and services decision")
        run("validate-production-gate-filled-log-ticket-linkage.py", filled)
        run("validate-production-gate-entered-filled-log-joint.py", joint)
        run("validate-r18-services-wide-slug-decision.py", services)
    hooks = link.get("runbook_checklist_hooks", [])
    if "C6" not in hooks or "main_table_8" not in hooks:
        raise SystemExit("runbook_checklist_hooks must include C6 and main_table_8")
    if status == "RUNBOOK_SECTION3_DEFER_ROLLUP_V149" and "C7" not in hooks:
        raise SystemExit("V149 runbook_checklist_hooks must include C7")
    if status == "RUNBOOK_SECTION3_DEFER_ROLLUP_V158":
        if "C10" not in hooks or "C13" not in hooks:
            raise SystemExit("V158 hooks must include C10 and C13")
    print("OK: runbook §3 defer rollup linkage chain")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()