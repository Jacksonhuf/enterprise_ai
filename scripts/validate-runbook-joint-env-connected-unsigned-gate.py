#!/usr/bin/env python3
"""Gate: CONNECTED shape rehearsal with unsigned UAT apply block (v1.71)."""
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
    if doc.get("gate_status") != "JOINT_ENV_CONNECTED_UNSIGNED_GATE_V171":
        raise SystemExit("gate_status must be JOINT_ENV_CONNECTED_UNSIGNED_GATE_V171")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-real-joint-environment-connected-shape-rehearsal.py"),
            str(FX / doc["connected_shape_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-unsigned-uat-apply-block-rehearsal.py"),
            str(FX / doc["unsigned_uat_apply_block_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-rehearsal-only-gate-enforcement.py"),
            str(FX / doc["rehearsal_only_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    gate = json.loads((FX / doc["rehearsal_only_gate_fixture"]).read_text())
    if gate.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match rehearsal-only gate")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: joint env CONNECTED shape + unsigned UAT gate (rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()