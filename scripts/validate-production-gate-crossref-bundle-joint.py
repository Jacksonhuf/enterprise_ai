#!/usr/bin/env python3
"""Validate crossref + bundle + ledger + staging joint warehouse rehearsal (v1.39)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"

ALLOWED = frozenset(
    {
        "CROSSREF_BUNDLE_JOINT_SHAPE_ONLY",
        "CROSSREF_BUNDLE_CONFIRMED_JOINT_SHAPE_ONLY",
    }
)


def main() -> None:
    path = Path(sys.argv[1])
    joint = json.loads(path.read_text())
    status = joint.get("joint_status")
    if status not in ALLOWED:
        raise SystemExit(f"joint_status must be one of {sorted(ALLOWED)}")
    ticket = joint.get("apply_change_ticket_ref", "")
    staging = joint.get("staging_import_ticket", "")
    if not ticket or not staging:
        raise SystemExit("apply_change_ticket_ref and staging_import_ticket required")
    if "REHEARSAL" in ticket.upper() or "REHEARSAL" in staging.upper():
        raise SystemExit("joint warehouse rehearsal must not use REHEARSAL ticket refs")
    xref = json.loads((FX / joint["crossref_fixture"]).read_text())
    if xref.get("crossref_status") != "WAREHOUSE_TICKET_SHAPE_ONLY":
        raise SystemExit("crossref_fixture must be WAREHOUSE_TICKET_SHAPE_ONLY")
    bundle = json.loads((FX / joint["bundle_fixture"]).read_text())
    ledger = json.loads((FX / joint["ledger_fixture"]).read_text())
    stix = json.loads((FX / joint["staging_ticket_fixture"]).read_text())
    for label, want, got in (
        ("crossref change", xref.get("apply_change_ticket_ref"), ticket),
        ("crossref staging", xref.get("staging_import_ticket"), staging),
        ("bundle change", bundle.get("apply_change_ticket_ref"), ticket),
        ("ledger change", ledger.get("apply_change_ticket_ref"), ticket),
        ("bundle staging", bundle.get("staging_import_ticket_ref"), staging),
        ("staging ticket", stix.get("staging_import_ticket"), staging),
    ):
        if got != want:
            raise SystemExit(f"{label} mismatch: expected {want}, got {got}")
    if ledger.get("ledger_status") not in ("ENTERED", "ENTERED_SHAPE_ONLY"):
        raise SystemExit("ledger must be ENTERED shape")
    bundle_status = bundle.get("bundle_status")
    if status == "CROSSREF_BUNDLE_CONFIRMED_JOINT_SHAPE_ONLY":
        if bundle_status != "CONFIRMED_SHAPE_ONLY":
            raise SystemExit("confirmed joint requires CONFIRMED_SHAPE_ONLY bundle")
    elif bundle_status not in ("APPROVED_SHAPE_ONLY", "CONFIRMED_SHAPE_ONLY"):
        raise SystemExit("bundle must be approved/confirmed shape")
    for field in ("confirmed_by_platform", "confirmed_by_sre", "verified_by_platform"):
        val = bundle.get(field) or ledger.get(field) or ""
        if val and "SIGNED" in val and "PLACEHOLDER" not in val:
            raise SystemExit(f"agent must not inject real signature in {field}")
    if status == "CROSSREF_BUNDLE_CONFIRMED_JOINT_SHAPE_ONLY":
        for field in ("confirmed_by_platform", "confirmed_by_sre"):
            if not bundle.get(field, "").startswith("PLACEHOLDER"):
                raise SystemExit(f"{field} must be PLACEHOLDER for shape rehearsal")
    print(f"OK: crossref-bundle joint ({status}) ({ticket} / {staging})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()