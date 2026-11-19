#!/usr/bin/env python3
"""Validate production gate ticket cross-reference table shape (v1.34)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"


def main() -> None:
    path = Path(sys.argv[1])
    xref = json.loads(path.read_text())
    if xref.get("crossref_status") != "CROSSREF_SHAPE_ONLY":
        raise SystemExit("crossref_status must be CROSSREF_SHAPE_ONLY")
    change = xref.get("apply_change_ticket_ref", "")
    staging = xref.get("staging_import_ticket", "")
    if not change or not staging:
        raise SystemExit("apply_change_ticket_ref and staging_import_ticket required")
    bundle = json.loads((FX / xref["bundle_fixture"]).read_text())
    ledger = json.loads((FX / xref["ledger_fixture"]).read_text())
    stix = json.loads((FX / xref["staging_ticket_fixture"]).read_text())
    if bundle.get("apply_change_ticket_ref") != change:
        raise SystemExit("bundle change ticket mismatch")
    if ledger.get("apply_change_ticket_ref") != change:
        raise SystemExit("ledger change ticket mismatch")
    if bundle.get("staging_import_ticket_ref") != staging:
        raise SystemExit("bundle staging ticket mismatch")
    if stix.get("staging_import_ticket") != staging:
        raise SystemExit("staging file ticket mismatch")
    print(f"OK: gate ticket crossref ({change} / {staging})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()