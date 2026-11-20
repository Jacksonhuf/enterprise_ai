#!/usr/bin/env python3
"""Production UAT signoff field shape (v1.67 rehearsal)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("shape_status") != "R02_PRODUCTION_UAT_SIGNOFF_SHAPE_V167":
        raise SystemExit("shape_status must be R02_PRODUCTION_UAT_SIGNOFF_SHAPE_V167")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    if doc.get("signoff_status") != "PENDING_UAT_TEAM_SIGN":
        raise SystemExit("signoff_status must be PENDING_UAT_TEAM_SIGN")
    for key in ("business_owner", "product_owner", "uat_lead_owner", "signed_at_cst"):
        if "PLACEHOLDER" not in str(doc.get(key, "")):
            raise SystemExit(f"{key} must be PLACEHOLDER in rehearsal")
    if not doc.get("production_apply_blocked"):
        raise SystemExit("production_apply_blocked must be true")
    pilot = ROOT / "docs/development" / doc.get("pilot_reference", "")
    if not pilot.is_file():
        raise SystemExit("missing pilot_reference doc")
    print("OK: production UAT signoff shape (rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()