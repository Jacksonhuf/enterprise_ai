#!/usr/bin/env python3
"""Gate: R22 UAT SLO crossref with R21 backup/SLO joint window (v1.78)."""
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
    if doc.get("gate_status") != "R22_JOINT_UAT_SLO_SIGNOFF_GATE_V178":
        raise SystemExit("gate_status must be R22_JOINT_UAT_SLO_SIGNOFF_GATE_V178")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-joint-uat-slo-signoff-crossref-ledger.py"),
            str(FX / doc["joint_uat_slo_crossref_ledger_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r21-backup-slo-evidence-joint-gate.py"),
            str(FX / doc["r21_backup_slo_joint_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    joint = json.loads((FX / doc["r21_backup_slo_joint_gate_fixture"]).read_text())
    if joint.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match backup/SLO joint gate")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: R22 joint UAT SLO signoff gate (mock)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()