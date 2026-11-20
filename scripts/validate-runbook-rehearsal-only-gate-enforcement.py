#!/usr/bin/env python3
"""Enforce rehearsal-only gate before real joint environment (v1.69)."""
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
    if doc.get("gate_status") != "REHEARSAL_ONLY_GATE_ENFORCEMENT_V169":
        raise SystemExit("gate_status must be REHEARSAL_ONLY_GATE_ENFORCEMENT_V169")
    if not doc.get("rehearsal_only_enforced"):
        raise SystemExit("rehearsal_only_enforced must be true")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-s5-release-ops-checklist-rehearsal.py"),
            str(FX / doc["s5_release_ops_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-real-joint-environment-hooklist-rehearsal.py"),
            str(FX / doc["joint_environment_hooklist_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-production-apply-preflight-triple-dry-run.py"),
            str(FX / doc["production_apply_preflight_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    preflight = json.loads((FX / doc["production_apply_preflight_fixture"]).read_text())
    if preflight.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match preflight fixture")
    print("OK: rehearsal-only gate enforcement (production apply forbidden)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()