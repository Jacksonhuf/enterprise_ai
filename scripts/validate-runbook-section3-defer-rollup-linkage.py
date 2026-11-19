#!/usr/bin/env python3
"""Validate runbook §3 checklist hooks against defer rollup and three-table window (v1.47)."""
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
    if link.get("linkage_status") != "RUNBOOK_SECTION3_DEFER_ROLLUP_V147":
        raise SystemExit("linkage_status must be RUNBOOK_SECTION3_DEFER_ROLLUP_V147")
    for doc in link.get("runbook_docs", []):
        if not (ROOT / "docs/development" / doc).is_file():
            raise SystemExit(f"missing runbook doc: {doc}")
    run("validate-r18-headroom-defer-rollup.py", link["defer_rollup_fixture"])
    run("validate-production-gate-three-table-window.py", link["three_table_window_fixture"])
    run("validate-staging-team-filled-rehearsal.py", link["staging_team_fill_fixture"])
    window = json.loads((FX / link["three_table_window_fixture"]).read_text())
    staging = json.loads((FX / link["staging_team_fill_fixture"]).read_text())
    if window.get("release_window_id") != staging.get("release_window_id"):
        raise SystemExit("team fill window must match three-table window id")
    hooks = link.get("runbook_checklist_hooks", [])
    if "C6" not in hooks or "main_table_8" not in hooks:
        raise SystemExit("runbook_checklist_hooks must include C6 and main_table_8")
    print("OK: runbook §3 defer rollup linkage chain")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()