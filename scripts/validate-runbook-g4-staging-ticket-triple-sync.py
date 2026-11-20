#!/usr/bin/env python3
"""Shape sync: G4 appendix, team-fill crossref, filled log change ticket (v1.61)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    link = json.loads(path.read_text())
    if link.get("sync_status") != "RUNBOOK_G4_STAGING_TICKET_TRIPLE_V161":
        raise SystemExit("sync_status must be RUNBOOK_G4_STAGING_TICKET_TRIPLE_V161")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-g4-filled-log-sha-linkage.py"), str(FX / link["g4_filled_log_linkage_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-production-gate-ticket-crossref-team-fill.py"), str(FX / link["crossref_fixture"])],
        check=True,
        cwd=ROOT,
    )
    xref = json.loads((FX / link["crossref_fixture"]).read_text())
    change = xref.get("apply_change_ticket_ref", "")
    log = json.loads((FX / link["filled_log_fixture"]).read_text())
    if log.get("change_ticket") != change:
        raise SystemExit("filled log change_ticket must match crossref")
    ex = link.get("triple_sync_shape_examples", {})
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-pilot-ticket-triple-sync.py"),
            str(FX / ex["bundle"]),
            str(FX / ex["ledger"]),
            str(FX / ex["staging"]),
        ],
        check=True,
        cwd=ROOT,
    )
    if link.get("g4_appendix_fixture") != "production-exporter-review-g4-appendix-rehearsal-v1.60.json":
        raise SystemExit("g4 appendix must be v1.60 rehearsal")
    print(f"OK: G4↔staging ticket triple sync shape ({change})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()