#!/usr/bin/env python3
"""Gate: R20 SLO team adoption with R22 dry-run window (v1.80)."""
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
    if doc.get("gate_status") != "R20_SLO_TEAM_ADOPTION_REHEARSAL_GATE_V180":
        raise SystemExit("gate_status must be R20_SLO_TEAM_ADOPTION_REHEARSAL_GATE_V180")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r20-slo-team-adoption-shape-ledger.py"),
            str(FX / doc["slo_team_adoption_ledger_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-uat-slo-crossref-dryrun-gate-rehearsal.py"),
            str(FX / doc["uat_slo_dryrun_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    dryrun = json.loads((FX / doc["uat_slo_dryrun_gate_fixture"]).read_text())
    if dryrun.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match dry-run gate")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: R20 SLO team adoption rehearsal gate (mock)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()