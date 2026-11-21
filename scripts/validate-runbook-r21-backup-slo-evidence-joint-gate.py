#!/usr/bin/env python3
"""Joint gate: R21 backup/RPO with R20 SLO evidence recheck (v1.77)."""
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
    if doc.get("gate_status") != "R21_BACKUP_SLO_EVIDENCE_JOINT_GATE_V177":
        raise SystemExit("gate_status must be R21_BACKUP_SLO_EVIDENCE_JOINT_GATE_V177")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r21-backup-rpo-slo-evidence-recheck-ledger.py"),
            str(FX / doc["backup_rpo_slo_recheck_ledger_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r21-backup-rpo-rto-rehearsal-gate.py"),
            str(FX / doc["backup_rpo_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r20-slo-evidence-rehearsal-gate.py"),
            str(FX / doc["slo_evidence_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    backup = json.loads((FX / doc["backup_rpo_gate_fixture"]).read_text())
    slo_ev = json.loads((FX / doc["slo_evidence_gate_fixture"]).read_text())
    window = doc.get("release_window_id")
    if backup.get("release_window_id") != window or slo_ev.get("release_window_id") != window:
        raise SystemExit("release_window_id must match nested gates")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: R21 backup/SLO evidence joint gate (mock)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()