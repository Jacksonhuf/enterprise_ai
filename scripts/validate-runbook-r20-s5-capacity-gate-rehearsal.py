#!/usr/bin/env python3
"""R20 measurement shape with S5 capacity gate rehearsal (v1.73)."""
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
    if doc.get("gate_status") != "R20_S5_CAPACITY_GATE_REHEARSAL_V173":
        raise SystemExit("gate_status must be R20_S5_CAPACITY_GATE_REHEARSAL_V173")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r20-pressure-sse-fault-measurement-shape.py"),
            str(FX / doc["measurement_shape_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-joint-env-shape-to-live-gate-rehearsal.py"),
            str(FX / doc["joint_env_shape_to_live_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    live = json.loads((FX / doc["joint_env_shape_to_live_gate_fixture"]).read_text())
    if live.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match shape-to-live gate")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: R20 S5 capacity gate rehearsal")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()