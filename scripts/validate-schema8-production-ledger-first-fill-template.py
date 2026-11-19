#!/usr/bin/env python3
"""Validate production ledger first real-fill pending template (v1.48)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if "FIRST_PRODUCTION_FILL_TEMPLATE_V148" not in data.get("notes", ""):
        raise SystemExit("notes must mark FIRST_PRODUCTION_FILL_TEMPLATE_V148")
    for key in ("production_cluster_id", "applied_at_cst", "verified_by_platform", "apply_change_ticket_ref", "runbook_smoke_summary_sha256"):
        if data.get(key):
            raise SystemExit(f"{key} must be empty in first-fill template")
    subprocess.run([PY, str(ROOT / "scripts/validate-production-ledger.py"), str(path)], check=True, cwd=ROOT)
    print("OK: first production fill ledger template")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()