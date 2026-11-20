#!/usr/bin/env python3
"""Validate R02 Figma field checklist shape and design ID coverage (v1.44)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MATRIX = ROOT / "docs/development/fixtures/r02-page-api-acceptance-matrix-v1.45.json"


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("checklist_status") not in (
        "R02_FIGMA_FIELD_CHECKLIST_V144",
        "R02_FIGMA_FIELD_CHECKLIST_V145",
        "R02_FIGMA_FIELD_CHECKLIST_V146",
        "R02_FIGMA_FIELD_CHECKLIST_V147",
        "R02_FIGMA_FIELD_CHECKLIST_V148",
        "R02_FIGMA_FIELD_CHECKLIST_V149",
        "R02_FIGMA_FIELD_CHECKLIST_V150",
        "R02_FIGMA_FIELD_CHECKLIST_V151",
        "R02_FIGMA_FIELD_CHECKLIST_V152",
    ):
        raise SystemExit("invalid checklist_status")
    if doc.get("signoff") not in (
        "REHEARSAL_PENDING",
        "REHEARSAL_PARTIAL",
        "PRODUCT_SIGNED",
    ):
        raise SystemExit("invalid signoff state")
    matched = sum(
        1
        for entry in doc.get("entries", [])
        for f in entry.get("fields", [])
        if f.get("status") == "matched"
    )
    if doc.get("checklist_status") == "R02_FIGMA_FIELD_CHECKLIST_V145" and matched < 1:
        raise SystemExit("v1.45 checklist requires at least one matched field")
    if doc.get("checklist_status") == "R02_FIGMA_FIELD_CHECKLIST_V146" and matched < 3:
        raise SystemExit("v1.46 checklist requires at least three matched fields")
    if doc.get("checklist_status") == "R02_FIGMA_FIELD_CHECKLIST_V147" and matched < 8:
        raise SystemExit("v1.47 checklist requires at least eight matched fields")
    if doc.get("checklist_status") == "R02_FIGMA_FIELD_CHECKLIST_V148" and matched < 11:
        raise SystemExit("v1.48 checklist requires at least eleven matched fields (half)")
    if doc.get("checklist_status") == "R02_FIGMA_FIELD_CHECKLIST_V149" and matched < 12:
        raise SystemExit("v1.49 checklist requires at least twelve matched fields")
    if doc.get("checklist_status") == "R02_FIGMA_FIELD_CHECKLIST_V150" and matched < 14:
        raise SystemExit("v1.50 checklist requires at least fourteen matched fields")
    if doc.get("checklist_status") == "R02_FIGMA_FIELD_CHECKLIST_V151" and matched < 17:
        raise SystemExit("v1.51 checklist requires at least seventeen matched fields")
    if doc.get("checklist_status") == "R02_FIGMA_FIELD_CHECKLIST_V152" and matched < 18:
        raise SystemExit("v1.52 checklist requires at least eighteen matched fields")
    entries = doc.get("entries", [])
    want = int(doc.get("design_entry_count", 17))
    if len(entries) != want:
        raise SystemExit(f"entries {len(entries)} != design_entry_count {want}")
    matrix_ids = {e["design_id"] for e in json.loads(MATRIX.read_text()).get("entries", [])}
    seen: set[str] = set()
    for entry in entries:
        did = entry.get("design_id")
        if did not in matrix_ids:
            raise SystemExit(f"checklist design_id not in matrix: {did}")
        if did in seen:
            raise SystemExit(f"duplicate design_id: {did}")
        seen.add(did)
        fields = entry.get("fields", [])
        if not fields:
            raise SystemExit(f"{did}: needs at least one field row")
        for f in fields:
            if f.get("status") not in ("pending", "matched", "waived"):
                raise SystemExit(f"{did}: invalid field status")
            if not f.get("id") or not f.get("figma_ref"):
                raise SystemExit(f"{did}: field needs id and figma_ref")
    if seen != matrix_ids:
        raise SystemExit(f"checklist IDs must match matrix exactly: missing {matrix_ids - seen}")
    print(f"OK: Figma field checklist ({len(entries)} entries)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()