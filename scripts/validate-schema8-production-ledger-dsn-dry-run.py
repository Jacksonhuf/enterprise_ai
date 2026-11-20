#!/usr/bin/env python3
"""§3 ledger DSN dry-run shape (v1.67); no production write."""
import json
import sys
from pathlib import Path


def main() -> None:
    path = Path(sys.argv[1])
    ledger = json.loads(path.read_text())
    if ledger.get("ledger_status") != "DRY_RUN_SHAPE_ONLY":
        raise SystemExit("ledger_status must be DRY_RUN_SHAPE_ONLY")
    if ledger.get("write_to_production_ledger") is not False:
        raise SystemExit("write_to_production_ledger must be false")
    dsn = str(ledger.get("target_dsn_shape", ""))
    if "PLACEHOLDER" not in dsn and "NOT-PRODUCTION" not in dsn:
        raise SystemExit("target_dsn_shape must be non-production placeholder")
    if "REHEARSAL" in dsn.upper():
        raise SystemExit("target_dsn_shape must use shape token not REHEARSAL prefix")
    ticket = ledger.get("apply_change_ticket_ref", "")
    if not ticket or ticket.startswith("REHEARSAL-"):
        raise SystemExit("apply_change_ticket_ref must be production-shape ticket")
    for field in ("verified_by_platform", "verified_by_sre"):
        if "PLACEHOLDER" not in str(ledger.get(field, "")):
            raise SystemExit(f"{field} must remain PLACEHOLDER")
    print(f"OK: DSN dry-run ledger shape ({ticket})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()