#!/usr/bin/env python3
"""Cross-check R02 gap closeouts against Figma matched fields (v1.51)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("crossref_status") != "R02_GAP_CLOSEOUT_FIGMA_V151":
        raise SystemExit("crossref_status must be R02_GAP_CLOSEOUT_FIGMA_V151")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-page-api-gap-closeout.py"), str(FX / doc["closeout_fixture"])],
        check=True,
        cwd=ROOT,
    )
    figma_path = FX / doc["figma_checklist_fixture"]
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-figma-field-checklist.py"), str(figma_path)],
        check=True,
        cwd=ROOT,
    )
    figma = json.loads(figma_path.read_text())
    if figma.get("checklist_status") != "R02_FIGMA_FIELD_CHECKLIST_V151":
        raise SystemExit("figma checklist must be V151")
    closeout = json.loads((FX / doc["closeout_fixture"]).read_text())
    required = doc.get("required_matched_fields", {})
    closeout_ids = {item["design_id"] for item in closeout.get("items", [])}
    if set(required.keys()) != closeout_ids:
        raise SystemExit("required_matched_fields keys must match closeout design_ids")
    matched_by_design: dict[str, set[str]] = {}
    for entry in figma.get("entries", []):
        did = entry["design_id"]
        matched_by_design[did] = {f["id"] for f in entry.get("fields", []) if f.get("status") == "matched"}
    for did, field_ids in required.items():
        got = matched_by_design.get(did, set())
        for fid in field_ids:
            if fid not in got:
                raise SystemExit(f"{did}: closeout requires matched figma field {fid}")
    print(f"OK: gap closeout↔figma crossref ({len(required)} designs)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()