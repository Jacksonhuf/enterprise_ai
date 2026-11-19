#!/usr/bin/env python3
"""Validate team-filled staging CLOSED ticket rehearsal (v1.47)."""
import json
import re
import sys
from pathlib import Path

FORBIDDEN = re.compile(r"PLACEHOLDER|SHAPE", re.I)


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("team_fill_status") != "TEAM_FILLED_REHEARSAL_V147":
        raise SystemExit("team_fill_status must be TEAM_FILLED_REHEARSAL_V147")
    if doc.get("ticket_status") != "CLOSED":
        raise SystemExit("ticket_status must be CLOSED (not CLOSED_SHAPE_ONLY)")
    if doc.get("ticket_status", "").endswith("_SHAPE_ONLY"):
        raise SystemExit("must not use *_SHAPE_ONLY for team-filled rehearsal")
    ticket = doc.get("staging_import_ticket", "")
    if not ticket or FORBIDDEN.search(ticket):
        raise SystemExit("staging_import_ticket required without PLACEHOLDER/SHAPE")
    for key in ("verified_by_sre", "imported_at_cst"):
        val = str(doc.get(key, ""))
        if not val:
            raise SystemExit(f"{key} required for team fill")
        if FORBIDDEN.search(val):
            raise SystemExit(f"{key} must not contain PLACEHOLDER/SHAPE")
    print(f"OK: team-filled staging rehearsal ({ticket})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()