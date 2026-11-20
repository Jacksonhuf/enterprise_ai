#!/usr/bin/env python3
"""R22 joint UAT scenario script rehearsal (v1.68)."""
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
    doc = json.loads(path.read_text())
    if doc.get("scenario_status") != "R22_JOINT_UAT_SCENARIO_V168":
        raise SystemExit("scenario_status must be R22_JOINT_UAT_SCENARIO_V168")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-production-uat-signoff-shape.py"), str(FX / doc["uat_signoff_shape_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-main-table-production-signoff-template.py"),
            str(FX / doc["main_table_signoff_template_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    pilot = DOC / doc.get("pilot_doc", "")
    if not pilot.is_file():
        raise SystemExit("missing pilot_doc")
    scenarios = doc.get("scenarios", [])
    if len(scenarios) != int(doc.get("expected_scenario_count", 0)):
        raise SystemExit("scenario count mismatch")
    matrix = json.loads((FX / "r02-page-api-acceptance-matrix-v1.45.json").read_text())
    design_ids = {e["design_id"] for e in matrix.get("entries", [])}
    personas = set()
    for sc in scenarios:
        if not sc.get("rehearsal_only"):
            raise SystemExit("each scenario must be rehearsal_only")
        if sc.get("matrix_design_id") not in design_ids:
            raise SystemExit(f"unknown design_id {sc.get('matrix_design_id')}")
        personas.add(sc.get("persona"))
    if len(personas) < 3:
        raise SystemExit("scenarios must cover at least three personas")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: R22 joint UAT scenarios ({len(scenarios)} rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()