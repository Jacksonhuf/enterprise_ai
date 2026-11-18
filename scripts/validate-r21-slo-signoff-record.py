#!/usr/bin/env python3
"""Validate R21 SLO signoff record JSON shape (v1.11); SIGNED requires owners."""
import json
import sys
from pathlib import Path

REQUIRED = frozenset(
    {
        "record_status",
        "pilot_reference",
        "adopted_targets_version",
        "platform_engineering_owner",
        "sre_owner",
        "business_owner",
        "signed_at_cst",
        "notes",
    }
)


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if set(data.keys()) != REQUIRED:
        raise SystemExit(f"keys must be {sorted(REQUIRED)}")
    if data["record_status"] == "SIGNED":
        for k in ("platform_engineering_owner", "sre_owner", "business_owner", "signed_at_cst"):
            if not data[k]:
                raise SystemExit(f"SIGNED record requires {k}")
    elif data["record_status"] != "PENDING_TEAM_SIGN":
        raise SystemExit("record_status must be PENDING_TEAM_SIGN or SIGNED")
    print(f"OK: slo signoff record ({data['record_status']})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()