#!/usr/bin/env python3
"""Validate first §3 ENTERED ledger rehearsal; signatory PLACEHOLDER gate (v1.57)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"


def main() -> None:
    path = Path(sys.argv[1])
    ledger = json.loads(path.read_text())
    if ledger.get("ledger_status") != "ENTERED":
        raise SystemExit("ledger_status must be ENTERED")
    if "FIRST_ENTERED_REHEARSAL_V157" not in ledger.get("notes", ""):
        raise SystemExit("notes must mark FIRST_ENTERED_REHEARSAL_V157")
    ticket = ledger.get("apply_change_ticket_ref", "")
    if not ticket or "PLACEHOLDER" in ticket:
        raise SystemExit("apply_change_ticket_ref must be set and not PLACEHOLDER")
    for field in ("verified_by_platform", "production_cluster_id", "applied_at_cst"):
        val = str(ledger.get(field, ""))
        if not val.startswith("PLACEHOLDER"):
            raise SystemExit(f"{field} must remain PLACEHOLDER gate for first ENTERED rehearsal")
    if ledger.get("verified_by_platform") == "":
        raise SystemExit("verified_by_platform required")
    print(f"OK: first ENTERED rehearsal ledger ({ticket})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()