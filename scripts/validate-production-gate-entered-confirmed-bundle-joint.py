#!/usr/bin/env python3
"""Validate ENTERED ledger + CONFIRMED bundle same ticket joint (v1.42)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"


def main() -> None:
    path = Path(sys.argv[1])
    joint = json.loads(path.read_text())
    if joint.get("joint_status") != "ENTERED_CONFIRMED_BUNDLE_JOINT_SHAPE_ONLY":
        raise SystemExit("joint_status must be ENTERED_CONFIRMED_BUNDLE_JOINT_SHAPE_ONLY")
    ticket = joint.get("apply_change_ticket_ref", "")
    staging = joint.get("staging_import_ticket", "")
    if not ticket or not staging:
        raise SystemExit("apply_change_ticket_ref and staging_import_ticket required")
    bundle = json.loads((FX / joint["bundle_fixture"]).read_text())
    ledger = json.loads((FX / joint["ledger_fixture"]).read_text())
    stix = json.loads((FX / joint["staging_ticket_fixture"]).read_text())
    if bundle.get("bundle_status") != "CONFIRMED":
        raise SystemExit("bundle must be CONFIRMED rehearsal")
    if ledger.get("ledger_status") != "ENTERED":
        raise SystemExit("ledger must be ENTERED")
    for label, want, got in (
        ("bundle change", bundle.get("apply_change_ticket_ref"), ticket),
        ("ledger change", ledger.get("apply_change_ticket_ref"), ticket),
        ("bundle staging", bundle.get("staging_import_ticket_ref"), staging),
        ("staging ticket", stix.get("staging_import_ticket"), staging),
    ):
        if got != want:
            raise SystemExit(f"{label} mismatch: expected {want}, got {got}")
    if "PLACEHOLDER" in ticket or "PLACEHOLDER" in staging:
        raise SystemExit("joint rehearsal tickets must not be PLACEHOLDER")
    print(f"OK: ENTERED/CONFIRMED bundle joint ({ticket} / {staging})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()