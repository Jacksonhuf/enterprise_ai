#!/usr/bin/env python3
"""Validate ENTERED ledger + APPROVED exporter + bundle joint rehearsal shape (v1.32)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FIXTURES = ROOT / "docs/development/fixtures"


def main() -> None:
    path = Path(sys.argv[1])
    joint = json.loads(path.read_text())
    if joint.get("joint_status") != "JOINT_SHAPE_ONLY":
        raise SystemExit("joint_status must be JOINT_SHAPE_ONLY")
    ticket = joint.get("apply_change_ticket_ref", "")
    if not ticket:
        raise SystemExit("apply_change_ticket_ref required")
    for key in ("ledger_fixture", "bundle_fixture", "exporter_review_fixture"):
        sub = FIXTURES / joint[key]
        if not sub.is_file():
            raise SystemExit(f"missing {key}: {sub}")
    ledger = json.loads((FIXTURES / joint["ledger_fixture"]).read_text())
    bundle = json.loads((FIXTURES / joint["bundle_fixture"]).read_text())
    exporter = json.loads((FIXTURES / joint["exporter_review_fixture"]).read_text())
    if ledger.get("ledger_status") != "ENTERED":
        raise SystemExit("ledger_fixture must be ENTERED shape")
    if ledger.get("apply_change_ticket_ref") != ticket:
        raise SystemExit("ledger apply_change_ticket_ref mismatch")
    if bundle.get("apply_change_ticket_ref") != ticket:
        raise SystemExit("bundle apply_change_ticket_ref mismatch")
    if exporter.get("checklist_status") not in ("APPROVED_SHAPE_ONLY", "APPROVED"):
        raise SystemExit("exporter must be APPROVED shape")
    if bundle.get("bundle_status") not in (
        "APPROVED_SHAPE_ONLY",
        "CONFIRMED_SHAPE_ONLY",
        "PENDING_TEAM_CONFIRM",
    ):
        raise SystemExit("unexpected bundle status for joint shape")
    print(f"OK: joint gate shape ({ticket})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()