#!/usr/bin/env python3
"""Validate bundle + ledger + staging + crossref share one release_window_id (v1.46)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"


def main() -> None:
    path = Path(sys.argv[1])
    win = json.loads(path.read_text())
    window_status = win.get("window_status")
    if window_status not in ("THREE_TABLE_WINDOW_SYNC_V146", "THREE_TABLE_WINDOW_SYNC_V148"):
        raise SystemExit("window_status must be THREE_TABLE_WINDOW_SYNC_V146 or V148")
    xref_status_by_window = {
        "THREE_TABLE_WINDOW_SYNC_V146": "THREE_TABLE_WINDOW_SHAPE_V146",
        "THREE_TABLE_WINDOW_SYNC_V148": "TEAM_FILL_STAGING_V148",
    }
    window_id = win.get("release_window_id", "")
    if not window_id:
        raise SystemExit("release_window_id required")
    xref = json.loads((FX / win["crossref_fixture"]).read_text())
    want_xref = xref_status_by_window[window_status]
    if xref.get("crossref_status") != want_xref:
        raise SystemExit(f"crossref must be {want_xref}")
    if xref.get("release_window_id") != window_id:
        raise SystemExit("crossref release_window_id mismatch")
    joint = json.loads((FX / win["entered_confirmed_joint_fixture"]).read_text())
    ticket = xref["apply_change_ticket_ref"]
    staging = xref["staging_import_ticket"]
    if joint.get("apply_change_ticket_ref") != ticket or joint.get("staging_import_ticket") != staging:
        raise SystemExit("entered/confirmed joint ticket mismatch crossref")
    bundle = json.loads((FX / xref["bundle_fixture"]).read_text())
    ledger = json.loads((FX / xref["ledger_fixture"]).read_text())
    stix = json.loads((FX / xref["staging_ticket_fixture"]).read_text())
    if stix.get("release_window_id") != window_id:
        raise SystemExit("staging release_window_id mismatch")
    if xref.get("release_window_id") != window_id:
        raise SystemExit("crossref release_window_id mismatch")
    for label, want, got in (
        ("bundle change", bundle.get("apply_change_ticket_ref"), ticket),
        ("ledger change", ledger.get("apply_change_ticket_ref"), ticket),
        ("bundle staging", bundle.get("staging_import_ticket_ref"), staging),
        ("staging ticket", stix.get("staging_import_ticket"), staging),
    ):
        if got != want:
            raise SystemExit(f"{label} mismatch: expected {want}, got {got}")
    if bundle.get("bundle_status") != "CONFIRMED":
        raise SystemExit("bundle must be CONFIRMED for window sync")
    if ledger.get("ledger_status") != "ENTERED":
        raise SystemExit("ledger must be ENTERED for window sync")
    if stix.get("ticket_status") != "CLOSED":
        raise SystemExit("staging ticket must be CLOSED for v1.46 window")
    print(f"OK: three-table window ({window_id})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()