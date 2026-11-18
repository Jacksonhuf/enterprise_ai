#!/usr/bin/env python3
"""Validate staging alert import ticket JSON (v1.10); closure requires team fields."""
import json
import sys
from pathlib import Path

REQUIRED = frozenset(
    {
        "ticket_status",
        "staging_import_ticket",
        "artifact_name",
        "rules_count",
        "imported_at_cst",
        "verified_by_sre",
        "notes",
    }
)


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if set(data.keys()) != REQUIRED:
        raise SystemExit(f"keys must be {sorted(REQUIRED)}")
    if data["artifact_name"] != "r21-staging-alert-render":
        raise SystemExit("artifact_name must be r21-staging-alert-render")
    if data["rules_count"] != 6:
        raise SystemExit("rules_count must be 6")
    if data["ticket_status"] == "CLOSED":
        if not data["imported_at_cst"] or not data["verified_by_sre"]:
            raise SystemExit("CLOSED ticket requires imported_at_cst and verified_by_sre")
    elif data["ticket_status"] != "PENDING_SRE_CLOSE":
        raise SystemExit("ticket_status must be PENDING_SRE_CLOSE or CLOSED")
    print(f"OK: staging import ticket ({data['ticket_status']})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()