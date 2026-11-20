#!/usr/bin/env python3
"""Joint environment hooks CONNECTED shape rehearsal (v1.71)."""
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
    if doc.get("shape_status") != "REAL_JOINT_ENVIRONMENT_CONNECTED_SHAPE_V171":
        raise SystemExit("shape_status must be REAL_JOINT_ENVIRONMENT_CONNECTED_SHAPE_V171")
    if doc.get("real_joint_environment_available"):
        raise SystemExit("real_joint_environment_available must stay false")
    if not doc.get("connected_shape_only"):
        raise SystemExit("connected_shape_only must be true")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-real-joint-environment-hooklist-rehearsal.py"),
            str(FX / doc["baseline_hooklist_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    baseline_ids = {
        h["hook_id"]
        for h in json.loads((FX / doc["baseline_hooklist_fixture"]).read_text()).get("hooks", [])
    }
    hooks = doc.get("hooks", [])
    if len(hooks) != int(doc.get("expected_hook_count", 0)):
        raise SystemExit("hook count mismatch")
    for hook in hooks:
        if hook.get("hook_id") not in baseline_ids:
            raise SystemExit(f"unknown hook_id {hook.get('hook_id')}")
        if hook.get("hook_state") != "CONNECTED_SHAPE":
            raise SystemExit(f"hook {hook.get('hook_id')} must be CONNECTED_SHAPE")
        if not hook.get("rehearsal_only"):
            raise SystemExit("each hook must be rehearsal_only")
        if "PLACEHOLDER" not in str(hook.get("shape_endpoint_ref", "")):
            raise SystemExit("shape_endpoint_ref must use PLACEHOLDER")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-real-human-uat-signoff-shape-rehearsal.py"),
            str(FX / doc["real_human_uat_signoff_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    signoff = json.loads((FX / doc["real_human_uat_signoff_fixture"]).read_text())
    if signoff.get("real_human_signoff_obtained"):
        raise SystemExit("human signoff must stay false in v1.71")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: joint environment CONNECTED shape ({len(hooks)} hooks, rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()