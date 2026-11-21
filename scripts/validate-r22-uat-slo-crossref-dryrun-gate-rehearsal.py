#!/usr/bin/env python3
"""Gate: R22 UAT dry-run recheck with v1.78 UAT/SLO gate (v1.79)."""
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
    if doc.get("gate_status") != "R22_UAT_SLO_CROSSREF_DRYRUN_GATE_V179":
        raise SystemExit("gate_status must be R22_UAT_SLO_CROSSREF_DRYRUN_GATE_V179")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-uat-slo-crossref-dryrun-recheck-ledger.py"),
            str(FX / doc["uat_slo_crossref_dryrun_recheck_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-joint-uat-slo-signoff-gate-rehearsal.py"),
            str(FX / doc["joint_uat_slo_signoff_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    parent = json.loads((FX / doc["joint_uat_slo_signoff_gate_fixture"]).read_text())
    if parent.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match v1.78 UAT/SLO gate")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: R22 UAT/SLO dry-run recheck gate (mock)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()