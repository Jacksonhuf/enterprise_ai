#!/usr/bin/env python3
"""Final pre-apply triple dry-run before production (v1.68)."""
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
    if doc.get("preflight_status") != "PRODUCTION_APPLY_PREFLIGHT_TRIPLE_V168":
        raise SystemExit("preflight_status must be PRODUCTION_APPLY_PREFLIGHT_TRIPLE_V168")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-production-gate-three-table-window.py"), str(FX / doc["three_table_window_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-dsn-dry-run-ledger-rehearsal.py"), str(FX / doc["dsn_dry_run_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-sre-platform-dual-signoff-shape.py"), str(FX / doc["dual_signoff_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-global-gap-item-closeout-rehearsal.py"), str(FX / doc["global_gap_closeout_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r22-joint-uat-scenario-rehearsal.py"), str(FX / doc["joint_uat_scenario_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-entered-ledger-write-gate.py"), str(FX / doc["entered_write_gate_fixture"])],
        check=True,
        cwd=ROOT,
    )
    win = json.loads((FX / doc["three_table_window_fixture"]).read_text())
    if win.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id mismatch")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: production apply preflight triple dry-run ({doc['release_window_id']})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()