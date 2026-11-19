#!/usr/bin/env python3
"""Reject production signature claims that still carry SHAPE or PLACEHOLDER markers (v1.46)."""
import json
import re
import sys
from pathlib import Path

FORBIDDEN = re.compile(r"SHAPE|PLACEHOLDER", re.I)
SIGNATURE_FIELDS = (
    "confirmed_by_platform",
    "confirmed_by_sre",
    "verified_by_platform",
    "verified_by_sre",
    "applied_by",
)


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    status = doc.get("bundle_status") or doc.get("ledger_status") or ""
    if status.endswith("_SHAPE_ONLY"):
        print(f"OK: explicit shape status, skip prod claim ({status})")
        print(f"Validated {path}")
        return
    claim = bool(doc.get("production_signature_claim"))
    non_shape_confirmed = status == "CONFIRMED"
    non_shape_entered = status == "ENTERED"
    if not claim and not (non_shape_confirmed or non_shape_entered):
        print(f"OK: no production signature claim on {path.name}")
        print(f"Validated {path}")
        return
    for key in SIGNATURE_FIELDS:
        val = doc.get(key)
        if not val:
            continue
        if FORBIDDEN.search(str(val)):
            raise SystemExit(f"{key} must not contain SHAPE/PLACEHOLDER when production claim applies: {val!r}")
    if non_shape_confirmed and "SHAPE" in status:
        raise SystemExit("CONFIRMED must not be CONFIRMED_SHAPE_ONLY for prod claim")
    print("OK: production signature claim passes shape rejection")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()