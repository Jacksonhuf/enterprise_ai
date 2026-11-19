#!/usr/bin/env python3
"""Ensure engineering pilot bundle and production ledger share change ticket ref (v1.31)."""
import json
import sys
from pathlib import Path

SHAPE_TICKET = "CHG-REHEARSAL-SHAPE-ONLY"


def main() -> None:
    if len(sys.argv) < 3:
        raise SystemExit("usage: validate-pilot-ledger-bundle-ticket-sync.py <bundle.json> <ledger.json>")
    bundle = json.loads(Path(sys.argv[1]).read_text())
    ledger = json.loads(Path(sys.argv[2]).read_text())
    bref = bundle.get("apply_change_ticket_ref", "")
    lref = ledger.get("apply_change_ticket_ref", "")
    if not bref or not lref:
        raise SystemExit("apply_change_ticket_ref required on bundle and ledger")
    if bref != lref:
        raise SystemExit(f"ticket ref mismatch: bundle={bref!r} ledger={lref!r}")
    real_pair = (
        bundle.get("bundle_status") == "CONFIRMED"
        and ledger.get("ledger_status") == "ENTERED"
    )
    if real_pair:
        for k in ("confirmed_by_platform", "confirmed_by_sre"):
            if "PLACEHOLDER" in bundle.get(k, ""):
                raise SystemExit(f"CONFIRMED bundle must not use PLACEHOLDER in {k}")
        for k in ("verified_by_platform", "production_cluster_id", "applied_at_cst"):
            if "PLACEHOLDER" in ledger.get(k, ""):
                raise SystemExit(f"ENTERED ledger must not use PLACEHOLDER in {k}")
    if bref == SHAPE_TICKET:
        print(f"OK: shape ticket sync ({SHAPE_TICKET})")
    else:
        print(f"OK: production ticket sync ({bref})")
    print(f"Validated {sys.argv[1]} + {sys.argv[2]}")


if __name__ == "__main__":
    main()