#!/usr/bin/env python3
"""Gate linking R20 SLO targets to rehearsal evidence (v1.76)."""
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
    if doc.get("gate_status") != "R20_SLO_EVIDENCE_REHEARSAL_GATE_V176":
        raise SystemExit("gate_status must be R20_SLO_EVIDENCE_REHEARSAL_GATE_V176")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r20-formal-slo-target-shape-ledger.py"),
            str(FX / doc["formal_slo_target_ledger_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r20-s5-capacity-gate-rehearsal.py"),
            str(FX / doc["r20_s5_capacity_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r21-oncall-handoff-gate-rehearsal.py"),
            str(FX / doc["r21_oncall_handoff_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    cap = json.loads((FX / doc["r20_s5_capacity_gate_fixture"]).read_text())
    oncall = json.loads((FX / doc["r21_oncall_handoff_gate_fixture"]).read_text())
    window = doc.get("release_window_id")
    if cap.get("release_window_id") != window or oncall.get("release_window_id") != window:
        raise SystemExit("release_window_id must match capacity and on-call gates")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: R20 SLO evidence rehearsal gate (mock)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()