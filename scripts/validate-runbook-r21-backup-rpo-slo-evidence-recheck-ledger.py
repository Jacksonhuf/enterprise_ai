#!/usr/bin/env python3
"""R21 backup/RPO + R20 SLO evidence same-window recheck ledger (v1.77)."""
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
    if doc.get("ledger_status") != "R21_BACKUP_SLO_EVIDENCE_RECHECK_V177":
        raise SystemExit("ledger_status must be R21_BACKUP_SLO_EVIDENCE_RECHECK_V177")
    if doc.get("joint_recheck_passed"):
        raise SystemExit("joint_recheck_passed must be false in rehearsal")
    for name in doc.get("pilot_docs", []):
        if not (DOC / name).is_file():
            raise SystemExit(f"missing pilot doc {name}")
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
        raise SystemExit("release_window_id must match backup and SLO evidence gates")
    ops = json.loads((FX / doc["ops_observability_ledger_fixture"]).read_text())
    slo_ledger = json.loads((FX / doc["formal_slo_target_ledger_fixture"]).read_text())
    if slo_ledger.get("formal_slo_adopted"):
        raise SystemExit("formal_slo_adopted must be false")
    for item in ops.get("ops_items", []):
        if item.get("item_id") == "rpo-rto-targets" and item.get("targets_adopted"):
            raise SystemExit("RPO/RTO targets_adopted must be false")
    items = doc.get("recheck_items", [])
    if len(items) != int(doc.get("expected_recheck_count", 0)):
        raise SystemExit("recheck_items count mismatch")
    for item in items:
        if item.get("recheck_executed"):
            raise SystemExit(f"{item.get('recheck_id')} recheck_executed must be false")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r21-slo-signoff-record.py"),
            str(FX / doc["slo_signoff_pending_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: R21 backup/SLO evidence recheck ledger ({len(items)} items, mock)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()