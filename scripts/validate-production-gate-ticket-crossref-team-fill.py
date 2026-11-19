#!/usr/bin/env python3
"""Validate crossref that pins team-filled CLOSED staging (v1.48)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    xref = json.loads(path.read_text())
    if xref.get("crossref_status") != "TEAM_FILL_STAGING_V148":
        raise SystemExit("crossref_status must be TEAM_FILL_STAGING_V148")
    staging_fixture = xref.get("staging_ticket_fixture", "")
    if "team-filled" not in staging_fixture:
        raise SystemExit("staging_ticket_fixture must reference team-filled rehearsal file")
    change = xref.get("apply_change_ticket_ref", "")
    staging = xref.get("staging_import_ticket", "")
    if not change or not staging:
        raise SystemExit("apply_change_ticket_ref and staging_import_ticket required")
    bundle = json.loads((FX / xref["bundle_fixture"]).read_text())
    ledger = json.loads((FX / xref["ledger_fixture"]).read_text())
    stix = json.loads((FX / staging_fixture).read_text())
    if bundle.get("apply_change_ticket_ref") != change:
        raise SystemExit("bundle change ticket mismatch")
    if ledger.get("apply_change_ticket_ref") != change:
        raise SystemExit("ledger change ticket mismatch")
    if bundle.get("staging_import_ticket_ref") != staging:
        raise SystemExit("bundle staging ticket mismatch")
    if stix.get("staging_import_ticket") != staging:
        raise SystemExit("staging file ticket mismatch")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-staging-team-filled-rehearsal.py"), str(FX / staging_fixture)],
        check=True,
        cwd=ROOT,
    )
    tmpl = xref.get("first_fill_ledger_template", "")
    if not tmpl:
        raise SystemExit("first_fill_ledger_template required for v1.48 crossref")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-schema8-production-ledger-first-fill-template.py"),
            str(FX / tmpl),
        ],
        check=True,
        cwd=ROOT,
    )
    print(f"OK: team-fill staging crossref ({change} / {staging})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()