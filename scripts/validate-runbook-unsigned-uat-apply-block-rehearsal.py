#!/usr/bin/env python3
"""Block production apply until real human UAT signoff (v1.70)."""
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
    if doc.get("block_status") != "UNSIGNED_UAT_PRODUCTION_APPLY_BLOCK_V170":
        raise SystemExit("block_status must be UNSIGNED_UAT_PRODUCTION_APPLY_BLOCK_V170")
    if not doc.get("unsigned_uat_apply_forbidden"):
        raise SystemExit("unsigned_uat_apply_forbidden must be true")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-real-human-uat-signoff-shape-rehearsal.py"),
            str(FX / doc["real_human_uat_signoff_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-rehearsal-only-gate-enforcement.py"),
            str(FX / doc["rehearsal_only_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    signoff = json.loads((FX / doc["real_human_uat_signoff_fixture"]).read_text())
    if signoff.get("real_human_signoff_obtained"):
        raise SystemExit("unsigned block requires real_human_signoff_obtained false")
    gate = json.loads((FX / doc["rehearsal_only_gate_fixture"]).read_text())
    if gate.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match rehearsal-only gate")
    print("OK: unsigned UAT production apply block (rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()