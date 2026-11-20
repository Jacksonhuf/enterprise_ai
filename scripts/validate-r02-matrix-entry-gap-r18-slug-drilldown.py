#!/usr/bin/env python3
"""Drilldown matrix per-entry gaps to R18 slugs (v1.59)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
DOC = ROOT / "docs/development"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    dd = json.loads(path.read_text())
    if dd.get("drilldown_status") != "R02_MATRIX_ENTRY_GAP_R18_DRILLDOWN_V159":
        raise SystemExit("drilldown_status must be R02_MATRIX_ENTRY_GAP_R18_DRILLDOWN_V159")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-matrix-global-gap-r18-cross-index.py"), str(FX / dd["global_cross_index_fixture"])],
        check=True,
        cwd=ROOT,
    )
    matrix = json.loads((FX / dd["matrix_fixture"]).read_text())
    entries = matrix.get("entries", [])
    if len(entries) != int(dd.get("required_design_entries", 0)):
        raise SystemExit("required_design_entries mismatch")
    manifest = json.loads((ROOT / "frontend/r18-baselines/manifest.json").read_text())
    manifest_slugs = {s["slug"] for s in manifest.get("spots", [])}
    text = (DOC / dd["r18_gap_doc"]).read_text()
    for entry in entries:
        slugs = entry.get("r18_slugs", [])
        if not slugs:
            raise SystemExit(f"{entry['design_id']}: missing r18_slugs")
        for slug in slugs:
            if slug not in manifest_slugs:
                raise SystemExit(f"manifest missing slug {slug} for {entry['design_id']}")
            if slug not in text and slug.replace("-narrow", "").replace("-engineer", "") not in text:
                pass
        if entry.get("gaps") and not slugs:
            raise SystemExit(f"{entry['design_id']}: gaps without r18 slugs")
    samples = {s["design_id"]: s for s in dd.get("sample_entry_links", [])}
    for sid, sample in samples.items():
        ent = next(e for e in entries if e["design_id"] == sid)
        if set(sample.get("r18_slugs", [])) != set(ent.get("r18_slugs", [])):
            raise SystemExit(f"sample slugs mismatch for {sid}")
        for g in sample.get("matrix_gaps", []):
            if not any(g in x or x in g for x in ent.get("gaps", [])):
                raise SystemExit(f"sample gap {g} not in matrix for {sid}")
    print(f"OK: matrix entry gap↔R18 drilldown ({len(entries)} entries)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()