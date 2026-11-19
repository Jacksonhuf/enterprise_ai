#!/usr/bin/env python3
"""Validate R02 matrix gap closeout registry against acceptance matrix (v1.49)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"

FORBIDDEN_AFTER_CLOSEOUT = {
    "CAND-10": frozenset({"TAB_PIXEL_GAP"}),
}


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("closeout_status") != "R02_GAP_CLOSEOUT_V149":
        raise SystemExit("closeout_status must be R02_GAP_CLOSEOUT_V149")
    matrix_path = FX / doc["matrix_fixture"]
    matrix = json.loads(matrix_path.read_text())
    entries_by_id = {e["design_id"]: e for e in matrix.get("entries", [])}
    for item in doc.get("items", []):
        did = item.get("design_id", "")
        code = item.get("gap_code", "")
        if did not in entries_by_id:
            raise SystemExit(f"unknown design_id in closeout: {did}")
        if not item.get("resolution"):
            raise SystemExit(f"{did}: resolution required")
        entry = entries_by_id[did]
        forbidden = FORBIDDEN_AFTER_CLOSEOUT.get(did, frozenset())
        if code in forbidden:
            gaps = entry.get("gaps", [])
            if code in gaps:
                raise SystemExit(f"{did}: gap {code} must be removed from matrix after closeout")
        slugs = entry.get("r18_slugs", [])
        if item.get("gap_code") == "DEDICATED_R18_SPOT" and did == "CAND-09":
            if "governance-access-groups" not in slugs:
                raise SystemExit("CAND-09 must list governance-access-groups slug")
        if item.get("gap_code") == "DEDICATED_R18_SPOT" and did == "CAND-11":
            if "governance-access-roles" not in slugs:
                raise SystemExit("CAND-11 must list governance-access-roles slug")
        if item.get("gap_code") == "DEDICATED_R18_SPOT" and did == "CAND-14":
            if "services-engineer" not in slugs:
                raise SystemExit("CAND-14 must list services-engineer slug")
    if matrix.get("matrix_status") != "R02_PAGE_API_ACCEPTANCE_MATRIX_V149":
        raise SystemExit("matrix must be R02_PAGE_API_ACCEPTANCE_MATRIX_V149")
    print(f"OK: R02 gap closeout ({len(doc.get('items', []))} items)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()