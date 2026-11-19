#!/usr/bin/env python3
"""Validate CONFIRMED (non-SHAPE) warehouse rehearsal bundle (v1.41)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
BUNDLE_SCRIPT = ROOT / "scripts/validate-engineering-pilot-bundle.py"


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if data.get("bundle_status") != "CONFIRMED":
        raise SystemExit("bundle_status must be CONFIRMED (non-SHAPE rehearsal)")
    if data.get("bundle_status", "").endswith("_SHAPE_ONLY"):
        raise SystemExit("must not use *_SHAPE_ONLY for this validator")
    subprocess.check_call([sys.executable, str(BUNDLE_SCRIPT), str(path)])
    for field in ("confirmed_by_platform", "confirmed_by_sre"):
        val = data[field]
        if "PLACEHOLDER" in val:
            raise SystemExit(f"{field}: CONFIRMED rehearsal must not use PLACEHOLDER")
        if "SIGNATORY" not in val or "WAREHOUSE" not in val:
            raise SystemExit(f"{field}: must be SIGNATORY-*-WAREHOUSE-* for rehearsal")
    for field in ("apply_change_ticket_ref", "staging_import_ticket_ref"):
        val = data[field]
        if "PLACEHOLDER" in val or not val:
            raise SystemExit(f"{field} required without PLACEHOLDER")
        if "REHEARSAL" not in val.upper() and "WAREHOUSE" not in val:
            raise SystemExit(f"{field} must be warehouse rehearsal ticket pattern")
    print("OK: CONFIRMED warehouse rehearsal bundle")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()