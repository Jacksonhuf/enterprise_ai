#!/usr/bin/env python3
"""Link filled REHEARSAL apply log ticket to three-table crossref (v1.57)."""
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
    if link.get("linkage_status") != "FILLED_LOG_TICKET_LINKAGE_V157":
        raise SystemExit("linkage_status must be FILLED_LOG_TICKET_LINKAGE_V157")
    if not link.get("not_production"):
        raise SystemExit("not_production must be true")
    log_path = FX / link["filled_log_fixture"]
    win_path = FX / link["three_table_window_fixture"]
    xref_path = FX / link["crossref_fixture"]
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(log_path)],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-production-gate-three-table-window.py"), str(win_path)],
        check=True,
        cwd=ROOT,
    )
    log = json.loads(log_path.read_text())
    xref = json.loads(xref_path.read_text())
    ticket = xref.get("apply_change_ticket_ref", "")
    if log.get("change_ticket") != ticket:
        raise SystemExit("filled log change_ticket must match crossref apply_change_ticket_ref")
    win = json.loads(win_path.read_text())
    if win.get("filled_log_ticket_linkage_fixture") != path.name:
        raise SystemExit("three-table window must reference this linkage fixture")
    print(f"OK: filled log ticket linkage ({ticket})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()