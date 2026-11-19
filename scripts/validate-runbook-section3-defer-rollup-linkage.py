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
    if status not in ("RUNBOOK_SECTION3_DEFER_ROLLUP_V147", "RUNBOOK_SECTION3_DEFER_ROLLUP_V149"):
        raise SystemExit("linkage_status must be V147 or V149")
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
    if status == "RUNBOOK_SECTION3_DEFER_ROLLUP_V149":
        xref_name = link.get("crossref_fixture", "")
        if not xref_name:
            raise SystemExit("crossref_fixture required for V149")
        run("validate-production-gate-ticket-crossref-team-fill.py", xref_name)
        tmpl = link.get("first_fill_ledger_template", "")
        if not tmpl:
            raise SystemExit("first_fill_ledger_template required for V149")
        run("validate-schema8-production-ledger-first-fill-template.py", tmpl)
        if link.get("three_table_window_fixture") != "production-gate-three-table-window-v1.48.json":
            raise SystemExit("V149 linkage must use production-gate-three-table-window-v1.48.json")
    hooks = link.get("runbook_checklist_hooks", [])
    if "C6" not in hooks or "main_table_8" not in hooks:
        raise SystemExit("runbook_checklist_hooks must include C6 and main_table_8")
    if status == "RUNBOOK_SECTION3_DEFER_ROLLUP_V149" and "C7" not in hooks:
        raise SystemExit("V149 runbook_checklist_hooks must include C7")
    print("OK: runbook §3 defer rollup linkage chain")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()