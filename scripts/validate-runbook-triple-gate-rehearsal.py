#!/usr/bin/env python3
"""Run triple ticket + joint gate rehearsal validators (v1.33)."""
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def run(cmd: list[str]) -> None:
    print("+", " ".join(cmd))
    subprocess.run(cmd, check=True, cwd=ROOT)


def main() -> None:
    run(
        [
            PY,
            "scripts/validate-pilot-ledger-bundle-ticket-sync.py",
            str(FX / "engineering-pilot-bundle-confirmed-shape-example.json"),
            str(FX / "schema8-production-ledger-entered-shape-example.json"),
        ]
    )
    run(
        [
            PY,
            "scripts/validate-pilot-ticket-triple-sync.py",
            str(FX / "engineering-pilot-bundle-approved-shape-example.json"),
            str(FX / "schema8-production-ledger-entered-shape-example.json"),
            str(FX / "staging-alert-import-ticket-closed-shape-example.json"),
        ]
    )
    run(
        [
            PY,
            "scripts/validate-production-gate-joint-shape.py",
            str(FX / "production-gate-entered-exporter-joint-shape-example.json"),
        ]
    )
    print("OK: runbook triple-gate rehearsal chain")


if __name__ == "__main__":
    main()