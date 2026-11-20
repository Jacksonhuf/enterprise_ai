#!/usr/bin/env python3
"""Real joint environment hook list rehearsal (v1.69)."""
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
    if doc.get("hooklist_status") != "REAL_JOINT_ENVIRONMENT_HOOKLIST_V169":
        raise SystemExit("hooklist_status must be REAL_JOINT_ENVIRONMENT_HOOKLIST_V169")
    if doc.get("real_joint_environment_available"):
        raise SystemExit("real_joint_environment_available must be false before joint UAT")
    hooks = doc.get("hooks", [])
    if len(hooks) != int(doc.get("expected_hook_count", 0)):
        raise SystemExit("hook count mismatch")
    for hook in hooks:
        if hook.get("hook_state") != "NOT_CONNECTED":
            raise SystemExit(f"hook {hook.get('hook_id')} must stay NOT_CONNECTED in rehearsal")
        if not hook.get("rehearsal_only"):
            raise SystemExit("each hook must be rehearsal_only")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-joint-uat-scenario-rehearsal.py"),
            str(FX / doc["joint_uat_scenario_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-s5-release-ops-checklist-rehearsal.py"),
            str(FX / doc["s5_release_ops_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: real joint environment hooklist ({len(hooks)} hooks, not connected)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()