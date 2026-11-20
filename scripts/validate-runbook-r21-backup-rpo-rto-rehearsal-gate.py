#!/usr/bin/env python3
"""Gate: R21 backup/RPO shape with R20 S5 capacity (v1.74)."""
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
    if doc.get("gate_status") != "R21_BACKUP_RPO_RTO_REHEARSAL_GATE_V174":
        raise SystemExit("gate_status must be R21_BACKUP_RPO_RTO_REHEARSAL_GATE_V174")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r21-ops-observability-shape-ledger.py"),
            str(FX / doc["ops_observability_ledger_fixture"]),
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
    r20 = json.loads((FX / doc["r20_s5_capacity_gate_fixture"]).read_text())
    if r20.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match R20 S5 gate")
    s5 = json.loads((FX / doc["s5_release_ops_fixture"]).read_text())
    if "R21" not in str(s5.get("task_ids", [])):
        raise SystemExit("S5 checklist must include R21")
    for item in s5.get("ops_items", []):
        if item.get("item_id") == "R21-deploy-observability" and item.get("exit_gate_met"):
            raise SystemExit("R21 S5 exit gate must stay false")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: R21 backup/RPO rehearsal gate")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()