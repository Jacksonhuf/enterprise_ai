#!/usr/bin/env python3
"""Run ordered runbook + R02 rehearsal validators (v1.53)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    chain = json.loads(path.read_text())
    chain_status = chain.get("chain_status")
    if chain_status not in (
        "RUNBOOK_REHEARSAL_CHAIN_V153",
        "RUNBOOK_REHEARSAL_CHAIN_V155",
        "RUNBOOK_REHEARSAL_CHAIN_V156",
        "RUNBOOK_REHEARSAL_CHAIN_V157",
        "RUNBOOK_REHEARSAL_CHAIN_V158",
        "RUNBOOK_REHEARSAL_CHAIN_V159",
        "RUNBOOK_REHEARSAL_CHAIN_V160",
        "RUNBOOK_REHEARSAL_CHAIN_V161",
        "RUNBOOK_REHEARSAL_CHAIN_V162",
        "RUNBOOK_REHEARSAL_CHAIN_V163",
        "RUNBOOK_REHEARSAL_CHAIN_V164",
        "RUNBOOK_REHEARSAL_CHAIN_V165",
        "RUNBOOK_REHEARSAL_CHAIN_V166",
        "RUNBOOK_REHEARSAL_CHAIN_V167",
        "RUNBOOK_REHEARSAL_CHAIN_V168",
        "RUNBOOK_REHEARSAL_CHAIN_V169",
        "RUNBOOK_REHEARSAL_CHAIN_V170",
        "RUNBOOK_REHEARSAL_CHAIN_V171",
        "RUNBOOK_REHEARSAL_CHAIN_V172",
        "RUNBOOK_REHEARSAL_CHAIN_V173",
        "RUNBOOK_REHEARSAL_CHAIN_V174",
        "RUNBOOK_REHEARSAL_CHAIN_V175",
        "RUNBOOK_REHEARSAL_CHAIN_V176",
        "RUNBOOK_REHEARSAL_CHAIN_V177",
        "RUNBOOK_REHEARSAL_CHAIN_V178",
        "RUNBOOK_REHEARSAL_CHAIN_V179",
        "RUNBOOK_REHEARSAL_CHAIN_V180",
    ):
        raise SystemExit("chain_status must be V153–V180")
    if not chain.get("not_production"):
        raise SystemExit("not_production must be true")
    steps = chain.get("steps", [])
    if len(steps) < 4:
        raise SystemExit("chain requires at least 4 steps")
    for step in steps:
        script = step.get("script", "")
        fixture = step.get("fixture", "")
        if not script or not fixture:
            raise SystemExit("each step needs script and fixture")
        target = ROOT / "scripts" / script
        if not target.is_file():
            raise SystemExit(f"missing script {script}")
        print(f"+ {script} {fixture}")
        subprocess.run([PY, str(target), str(FX / fixture)], check=True, cwd=ROOT)
    print(f"OK: rehearsal chain orchestration ({len(steps)} steps)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()