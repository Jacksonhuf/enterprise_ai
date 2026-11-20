#!/usr/bin/env python3
"""Joint gate: global gap dashboard + matrix↔figma bidirectional (v1.56)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
DOC = ROOT / "docs/development"
PY = sys.executable


def run(script: str, fixture: str) -> None:
    subprocess.run([PY, str(ROOT / "scripts" / script), str(FX / fixture)], check=True, cwd=ROOT)


def main() -> None:
    path = Path(sys.argv[1])
    joint = json.loads(path.read_text())
    status = joint.get("joint_status")
    if status not in (
        "R02_GLOBAL_GAP_DASHBOARD_MATRIX_JOINT_V156",
        "R02_GLOBAL_GAP_DASHBOARD_MATRIX_JOINT_V157",
    ):
        raise SystemExit("joint_status must be V156 or V157")
    if not joint.get("not_production"):
        raise SystemExit("not_production must be true")
    viz = joint.get("global_gap_visual_fixture", "")
    bidir = joint.get("matrix_bidirectional_fixture", "")
    reg = joint.get("open_gap_registry_fixture", "")
    if not viz or not bidir or not reg:
        raise SystemExit("viz, bidirectional, and registry fixtures required")
    run("validate-r02-matrix-global-gap-visualization.py", viz)
    run("validate-r02-matrix-figma-bidirectional.py", bidir)
    run("validate-r02-page-api-open-gap-registry.py", reg)
    viz_doc = json.loads((FX / viz).read_text())
    if viz_doc.get("dashboard_matrix_joint_fixture") != path.name:
        raise SystemExit("viz must point back to this joint fixture")
    figma_path = FX / joint["figma_checklist_fixture"]
    run("validate-r02-figma-field-checklist.py", joint["figma_checklist_fixture"])
    figma = json.loads(figma_path.read_text())
    want_signoff = joint.get("rules", {}).get("engineering_field_signoff")
    if figma.get("signoff") != want_signoff:
        raise SystemExit(f"figma signoff must be {want_signoff}")
    matrix = json.loads((FX / json.loads((FX / viz).read_text())["matrix_fixture"]).read_text())
    blocker = joint.get("rules", {}).get("global_blocker_must_include")
    if blocker not in matrix.get("r18_global_gaps", []):
        raise SystemExit(f"matrix must still list global blocker {blocker}")
    doc_name = joint.get("figma_blocker_doc", "")
    doc_path = DOC / doc_name
    if not doc_path.is_file():
        raise SystemExit(f"missing figma blocker doc {doc_name}")
    text = doc_path.read_text()
    for needle in ("FIGMA_SIGNOFF_ALL", "REHEARSAL_ALL_FIELDS_MATCHED", "非产品签字"):
        if needle not in text:
            raise SystemExit(f"figma blocker doc must mention {needle}")
    cross = joint.get("r18_gap_cross_index_fixture", "")
    if status == "R02_GLOBAL_GAP_DASHBOARD_MATRIX_JOINT_V157":
        if not cross:
            raise SystemExit("V157 joint requires r18_gap_cross_index_fixture")
        run("validate-r02-matrix-global-gap-r18-cross-index.py", cross)
    print("OK: global gap↔dashboard↔matrix joint gate")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()