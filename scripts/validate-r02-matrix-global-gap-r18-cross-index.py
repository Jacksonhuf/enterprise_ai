#!/usr/bin/env python3
"""Cross-index matrix global gaps with R18 gap checklist doc (v1.57)."""
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
    idx = json.loads(path.read_text())
    if idx.get("index_status") != "R02_MATRIX_GLOBAL_GAP_R18_CROSS_INDEX_V157":
        raise SystemExit("index_status must be R02_MATRIX_GLOBAL_GAP_R18_CROSS_INDEX_V157")
    viz = idx.get("global_gap_visual_fixture", "")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-matrix-global-gap-visualization.py"), str(FX / viz)],
        check=True,
        cwd=ROOT,
    )
    viz_doc = json.loads((FX / viz).read_text())
    if viz_doc.get("r18_gap_cross_index_fixture") != path.name:
        raise SystemExit("viz must point back to this cross-index fixture")
    matrix = json.loads((FX / idx["matrix_fixture"]).read_text())
    global_gaps = set(matrix.get("r18_global_gaps", []))
    doc_path = DOC / idx["r18_gap_doc"]
    if not doc_path.is_file():
        raise SystemExit(f"missing r18 gap doc {idx['r18_gap_doc']}")
    text = doc_path.read_text()
    mapped = {m["matrix_gap"] for m in idx.get("global_mappings", [])}
    if mapped != global_gaps:
        raise SystemExit(f"global_mappings must cover exactly {sorted(global_gaps)}")
    for m in idx.get("global_mappings", []):
        if m["doc_anchor"] not in text:
            raise SystemExit(f"r18 doc must mention anchor {m['doc_anchor']}")
        if not m.get("r18_enum"):
            raise SystemExit("each mapping needs r18_enum")
    print(f"OK: matrix global gap↔R18 cross-index ({len(mapped)} gaps)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()