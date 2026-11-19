#!/usr/bin/env python3
"""Validate production apply ledger JSON (v1.14)."""
import json
import sys
from pathlib import Path

REQUIRED = frozenset(
    {
        "ledger_status",
        "apply_runbook_section",
        "production_cluster_id",
        "applied_at_cst",
        "verified_by_platform",
        "runbook_smoke_summary_artifact",
        "runbook_smoke_summary_sha256",
        "apply_change_ticket_ref",
        "notes",
    }
)


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if set(data.keys()) != REQUIRED:
        raise SystemExit(f"keys must be {sorted(REQUIRED)}")
    if data["apply_runbook_section"] != "3":
        raise SystemExit("apply_runbook_section must be 3")
    if data["ledger_status"] == "ENTERED":
        if not data["production_cluster_id"] or not data["applied_at_cst"] or not data["verified_by_platform"]:
            raise SystemExit("ENTERED ledger requires cluster_id, applied_at_cst, verified_by_platform")
        sha = data["runbook_smoke_summary_sha256"]
        if not sha or (sha != "PLACEHOLDER-NOT-PRODUCTION" and len(sha) != 64):
            raise SystemExit("ENTERED ledger requires smoke_summary_sha256 (PLACEHOLDER or 64 hex)")
        if not data["apply_change_ticket_ref"]:
            raise SystemExit("ENTERED ledger requires apply_change_ticket_ref")
    elif data["ledger_status"] != "PENDING_TEAM_ENTRY":
        raise SystemExit("ledger_status must be PENDING_TEAM_ENTRY or ENTERED")
    print(f"OK: production ledger ({data['ledger_status']})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()