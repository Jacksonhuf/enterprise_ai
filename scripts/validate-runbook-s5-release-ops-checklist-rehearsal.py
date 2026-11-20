#!/usr/bin/env python3
"""S5 release operations checklist rehearsal (v1.69)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
DOC = ROOT / "docs/development"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("checklist_status") != "RUNBOOK_S5_RELEASE_OPS_CHECKLIST_V169":
        raise SystemExit("checklist_status must be RUNBOOK_S5_RELEASE_OPS_CHECKLIST_V169")
    pilot = DOC / doc.get("delivery_plan_doc", "")
    if not pilot.is_file():
        raise SystemExit("missing delivery_plan_doc")
    task_ids = doc.get("task_ids", [])
    if task_ids != ["R20", "R21", "R22"]:
        raise SystemExit("task_ids must be R20, R21, R22")
    items = doc.get("ops_items", [])
    if len(items) != 3:
        raise SystemExit("ops_items must list three S5 tasks")
    for item in items:
        if item.get("exit_gate_met"):
            raise SystemExit("S5 exit gates must stay false in rehearsal")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-production-apply-preflight-triple-dry-run.py"),
            str(FX / doc["production_apply_preflight_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-joint-uat-scenario-rehearsal.py"),
            str(FX / doc["joint_uat_scenario_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    win = json.loads((FX / doc["triple_gate_window_fixture"]).read_text())
    window_id = str(win.get("release_window_id", ""))
    if "REHEARSAL" not in window_id:
        raise SystemExit("triple gate window must stay rehearsal release_window_id")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: S5 release ops checklist ({len(items)} items, rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()