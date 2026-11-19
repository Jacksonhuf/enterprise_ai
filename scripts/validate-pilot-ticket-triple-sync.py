#!/usr/bin/env python3
"""Bundle + ledger change ticket and bundle + staging import ticket alignment (v1.32)."""
import json
import sys
from pathlib import Path


def main() -> None:
    if len(sys.argv) < 4:
        raise SystemExit(
            "usage: validate-pilot-ticket-triple-sync.py <bundle.json> <ledger.json> <staging-ticket.json>"
        )
    bundle = json.loads(Path(sys.argv[1]).read_text())
    ledger = json.loads(Path(sys.argv[2]).read_text())
    staging = json.loads(Path(sys.argv[3]).read_text())
    b_change = bundle.get("apply_change_ticket_ref", "")
    l_change = ledger.get("apply_change_ticket_ref", "")
    b_staging = bundle.get("staging_import_ticket_ref", "")
    s_staging = staging.get("staging_import_ticket", "")
    if not b_change or not l_change:
        raise SystemExit("apply_change_ticket_ref required on bundle and ledger")
    if b_change != l_change:
        raise SystemExit(f"change ticket mismatch: bundle={b_change!r} ledger={l_change!r}")
    if b_staging and s_staging and b_staging != s_staging:
        raise SystemExit(f"staging ticket mismatch: bundle={b_staging!r} staging={s_staging!r}")
    # When bundle links staging, staging ticket file must be present
    if b_staging and not s_staging:
        raise SystemExit("staging ticket file missing staging_import_ticket")
    print(f"OK: triple ticket sync (change={b_change!r}, staging={b_staging or 'n/a'!r})")
    print(f"Validated {sys.argv[1]} + {sys.argv[2]} + {sys.argv[3]}")


if __name__ == "__main__":
    main()