#!/usr/bin/env python3
"""Validate R02 matrix V149 and Figma checklist bidirectional mapping (v1.50)."""
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
    link_status = link.get("link_status")
    if link_status not in ("R02_MATRIX_FIGMA_BIDIRECTIONAL_V150", "R02_MATRIX_FIGMA_BIDIRECTIONAL_V151"):
        raise SystemExit("link_status must be V150 or V151")
    matrix_path = FX / link["matrix_fixture"]
    figma_path = FX / link["figma_checklist_fixture"]
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-page-api-acceptance-matrix.py"), str(matrix_path)],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-figma-field-checklist.py"), str(figma_path)],
        check=True,
        cwd=ROOT,
    )
    matrix = json.loads(matrix_path.read_text())
    figma = json.loads(figma_path.read_text())
    if matrix.get("matrix_status") not in (
        "R02_PAGE_API_ACCEPTANCE_MATRIX_V149",
        "R02_PAGE_API_ACCEPTANCE_MATRIX_V150",
    ):
        raise SystemExit("matrix must be V149 or V150")
    figma_status = figma.get("checklist_status")
    if figma_status not in (
        "R02_FIGMA_FIELD_CHECKLIST_V150",
        "R02_FIGMA_FIELD_CHECKLIST_V151",
        "R02_FIGMA_FIELD_CHECKLIST_V152",
        "R02_FIGMA_FIELD_CHECKLIST_V153",
        "R02_FIGMA_FIELD_CHECKLIST_V154",
    ):
        raise SystemExit("figma checklist must be V150, V151, or V152")
    matrix_ids = {e["design_id"] for e in matrix.get("entries", [])}
    checklist_by_id = {e["design_id"]: e for e in figma.get("entries", [])}
    if set(matrix_ids) != set(checklist_by_id):
        raise SystemExit("matrix and figma checklist design_id sets differ")
    total_matched = 0
    for entry in matrix.get("entries", []):
        did = entry["design_id"]
        gaps = entry.get("gaps", [])
        fields = checklist_by_id[did].get("fields", [])
        if any("FIGMA" in g for g in gaps) and not fields:
            raise SystemExit(f"{did}: FIGMA gap in matrix but no checklist fields")
        mcount = sum(1 for f in fields if f.get("status") == "matched")
        total_matched += mcount
    min_by_status = {
        "R02_FIGMA_FIELD_CHECKLIST_V150": 14,
        "R02_FIGMA_FIELD_CHECKLIST_V151": 17,
        "R02_FIGMA_FIELD_CHECKLIST_V152": 18,
        "R02_FIGMA_FIELD_CHECKLIST_V153": 19,
        "R02_FIGMA_FIELD_CHECKLIST_V154": 20,
    }
    min_matched = min_by_status.get(figma_status, 14)
    if total_matched < min_matched:
        raise SystemExit(f"bidirectional link requires at least {min_matched} matched figma fields")
    xref = link.get("rules", {}).get("gap_closeout_figma_crossref_fixture")
    if link_status == "R02_MATRIX_FIGMA_BIDIRECTIONAL_V151":
        if not xref:
            raise SystemExit("V151 link must name gap_closeout_figma_crossref_fixture")
        subprocess.run(
            [PY, str(ROOT / "scripts/validate-r02-gap-closeout-figma-crossref.py"), str(FX / xref)],
            check=True,
            cwd=ROOT,
        )
    print(f"OK: matrix↔figma bidirectional ({total_matched} matched fields)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()