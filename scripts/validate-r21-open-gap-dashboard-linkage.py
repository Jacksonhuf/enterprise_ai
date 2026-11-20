#!/usr/bin/env python3
"""Link R21 rehearsal chain with R02 open-gap dashboard and Figma plan (v1.53)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def run(script: str, fixture: str) -> None:
    subprocess.run([PY, str(ROOT / "scripts" / script), str(FX / fixture)], check=True, cwd=ROOT)


def main() -> None:
    path = Path(sys.argv[1])
    link = json.loads(path.read_text())
    link_status = link.get("linkage_status")
    if link_status not in (
        "R21_OPEN_GAP_DASHBOARD_V153",
        "R21_OPEN_GAP_DASHBOARD_V154",
        "R21_OPEN_GAP_DASHBOARD_V155",
        "R21_OPEN_GAP_DASHBOARD_V156",
    ):
        raise SystemExit("linkage_status must be V153–V156")
    for doc in link.get("runbook_docs", []):
        if not (ROOT / "docs/development" / doc).is_file():
            raise SystemExit(f"missing doc {doc}")
    run("validate-r02-page-api-open-gap-registry.py", link["open_gap_registry_fixture"])
    run("validate-runbook-rehearsal-chain-orchestration.py", link["rehearsal_chain_fixture"])
    run("validate-runbook-first-fill-rehearsal-table.py", link["first_fill_rehearsal_fixture"])
    run("validate-r02-figma-pending-closeout-plan.py", link["figma_pending_plan_fixture"])
    viz = link.get("global_gap_visual_fixture")
    if link_status in (
        "R21_OPEN_GAP_DASHBOARD_V154",
        "R21_OPEN_GAP_DASHBOARD_V155",
        "R21_OPEN_GAP_DASHBOARD_V156",
    ):
        if not viz:
            raise SystemExit("V154+ linkage requires global_gap_visual_fixture")
        run("validate-r02-matrix-global-gap-visualization.py", viz)
    chain_fixture = link.get("rehearsal_chain_fixture", "")
    if link_status == "R21_OPEN_GAP_DASHBOARD_V155":
        if "v1.55" not in chain_fixture:
            raise SystemExit("V155 dashboard must use v1.55 rehearsal chain")
        run("validate-runbook-rehearsal-chain-orchestration.py", chain_fixture)
    if link_status == "R21_OPEN_GAP_DASHBOARD_V156":
        if "v1.56" not in chain_fixture:
            raise SystemExit("V156 dashboard must use v1.56 rehearsal chain")
        joint = link.get("dashboard_matrix_joint_fixture", "")
        if not joint:
            raise SystemExit("V156 linkage requires dashboard_matrix_joint_fixture")
        if "r02-global-figma-blocker-v1.56.md" not in link.get("runbook_docs", []):
            raise SystemExit("V156 dashboard must list figma blocker doc")
        run("validate-r02-global-gap-dashboard-matrix-joint.py", joint)
    print("OK: R21 open-gap dashboard linkage")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()