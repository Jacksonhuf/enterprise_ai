#!/usr/bin/env python3
"""Run shape-rejection gate: negative fixtures must fail, positives must pass (v1.46)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
CHECK = ROOT / "scripts/validate-production-gate-rejects-shape-signatures.py"


def run(path: Path) -> int:
    return subprocess.call([sys.executable, str(CHECK), str(path)])


def main() -> None:
    path = Path(sys.argv[1])
    gate = json.loads(path.read_text())
    if gate.get("gate_status") != "SHAPE_REJECTION_GATE_V146":
        raise SystemExit("gate_status must be SHAPE_REJECTION_GATE_V146")
    for name in gate.get("reject_fixtures", []):
        fp = FX / name
        if run(fp) == 0:
            raise SystemExit(f"expected rejection for {name}")
    for name in gate.get("accept_fixtures", []):
        fp = FX / name
        if run(fp) != 0:
            raise SystemExit(f"expected acceptance for {name}")
    print(f"OK: shape-rejection gate ({len(gate.get('reject_fixtures', []))} reject, "
          f"{len(gate.get('accept_fixtures', []))} accept)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()