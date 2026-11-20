#!/usr/bin/env python3
"""R21 on-call handoff gate rehearsal (v1.75)."""
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
    if doc.get("gate_status") != "R21_ONCALL_HANDOFF_GATE_V175":
        raise SystemExit("gate_status must be R21_ONCALL_HANDOFF_GATE_V175")
    if doc.get("handoff_recorded"):
        raise SystemExit("handoff_recorded must be false in rehearsal")
    for key in ("oncall_primary_owner", "oncall_secondary_owner"):
        if "PLACEHOLDER" not in str(doc.get(key, "")):
            raise SystemExit(f"{key} must be PLACEHOLDER")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r21-alert-dryrun-shape-ledger.py"),
            str(FX / doc["alert_dryrun_ledger_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r21-backup-rpo-rto-rehearsal-gate.py"),
            str(FX / doc["r21_backup_rpo_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    backup = json.loads((FX / doc["r21_backup_rpo_gate_fixture"]).read_text())
    if backup.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match backup gate")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: R21 on-call handoff gate (rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()