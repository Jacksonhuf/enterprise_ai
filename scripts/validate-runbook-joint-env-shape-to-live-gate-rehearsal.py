#!/usr/bin/env python3
"""Gate: CONNECTED shape to LIVE hooks requires separate approval (v1.72)."""
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
    if doc.get("gate_status") != "JOINT_ENV_SHAPE_TO_LIVE_GATE_V172":
        raise SystemExit("gate_status must be JOINT_ENV_SHAPE_TO_LIVE_GATE_V172")
    if doc.get("live_hooks_permitted"):
        raise SystemExit("live_hooks_permitted must be false in rehearsal")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-real-joint-environment-connected-shape-rehearsal.py"),
            str(FX / doc["connected_shape_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    shape = json.loads((FX / doc["connected_shape_fixture"]).read_text())
    prior = doc.get("required_prior_hook_state", "")
    for hook in shape.get("hooks", []):
        if hook.get("hook_state") != prior:
            raise SystemExit("all hooks must be CONNECTED_SHAPE before LIVE gate")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-joint-env-connected-unsigned-gate.py"),
            str(FX / doc["joint_env_connected_unsigned_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-real-human-uat-signoff-dryrun-ledger.py"),
            str(FX / doc["uat_signoff_dryrun_ledger_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    unsigned = json.loads((FX / doc["joint_env_connected_unsigned_gate_fixture"]).read_text())
    if unsigned.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match connected unsigned gate")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: joint env shape-to-LIVE gate (LIVE not permitted)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()