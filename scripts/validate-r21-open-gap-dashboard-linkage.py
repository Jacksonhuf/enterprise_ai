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
        "R21_OPEN_GAP_DASHBOARD_V157",
        "R21_OPEN_GAP_DASHBOARD_V158",
        "R21_OPEN_GAP_DASHBOARD_V159",
        "R21_OPEN_GAP_DASHBOARD_V160",
        "R21_OPEN_GAP_DASHBOARD_V161",
        "R21_OPEN_GAP_DASHBOARD_V162",
        "R21_OPEN_GAP_DASHBOARD_V163",
        "R21_OPEN_GAP_DASHBOARD_V164",
    ):
        raise SystemExit("linkage_status must be V153–V164")
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
    if link_status == "R21_OPEN_GAP_DASHBOARD_V157":
        if "v1.57" not in chain_fixture:
            raise SystemExit("V157 dashboard must use v1.57 rehearsal chain")
        joint = link.get("dashboard_matrix_joint_fixture", "")
        if "v1.57" not in joint:
            raise SystemExit("V157 dashboard must use joint v1.57")
        if "r18-gap-checklist-v1.43.md" not in link.get("runbook_docs", []):
            raise SystemExit("V157 dashboard must list r18 gap checklist doc")
        win = link.get("three_table_window_fixture", "")
        if not win:
            raise SystemExit("V157 dashboard requires three_table_window_fixture")
        run("validate-production-gate-three-table-window.py", win)
    if link_status == "R21_OPEN_GAP_DASHBOARD_V158":
        if "v1.58" not in chain_fixture:
            raise SystemExit("V158 dashboard must use v1.58 rehearsal chain")
        defer = link.get("defer_rollup_linkage_fixture", "")
        if "v1.58" not in defer:
            raise SystemExit("V158 dashboard must use defer linkage v1.58")
        run("validate-runbook-section3-defer-rollup-linkage.py", defer)
    if link_status == "R21_OPEN_GAP_DASHBOARD_V159":
        if "v1.59" not in chain_fixture:
            raise SystemExit("V159 dashboard must use v1.59 rehearsal chain")
        reg = link.get("triple_gate_window_registry_fixture", "")
        drill = link.get("matrix_entry_drilldown_fixture", "")
        if not reg or not drill:
            raise SystemExit("V159 dashboard must name registry and drilldown fixtures")
        if "mysql-schema8-runbook-production-fill-checklist.md" not in link.get("runbook_docs", []):
            raise SystemExit("V159 dashboard must list production fill checklist doc")
        run("validate-r21-dashboard-triple-gate-window-registry.py", reg)
        run("validate-r02-matrix-entry-gap-r18-slug-drilldown.py", drill)
    if link_status == "R21_OPEN_GAP_DASHBOARD_V160":
        if "v1.60" not in chain_fixture:
            raise SystemExit("V160 dashboard must use v1.60 rehearsal chain")
        reg = link.get("triple_gate_window_registry_fixture", "")
        if "v1.60" not in reg:
            raise SystemExit("V160 dashboard must use triple-gate registry v1.60")
        g4 = link.get("g4_filled_log_linkage_fixture", "")
        groups = link.get("groups_roles_defer_fixture", "")
        if not g4 or not groups:
            raise SystemExit("V160 dashboard must name G4 and groups/roles fixtures")
        if "mysql-schema8-runbook-section3-g4-linkage.md" not in link.get("runbook_docs", []):
            raise SystemExit("V160 dashboard must list G4 linkage doc")
        run("validate-r21-dashboard-triple-gate-window-registry.py", reg)
        run("validate-runbook-g4-filled-log-sha-linkage.py", g4)
        run("validate-r18-groups-roles-tab-defer-decision.py", groups)
        run("validate-runbook-main-table8-dashboard-window-check.py", "runbook-main-table8-dashboard-window-check-v1.60.json")
    if link_status == "R21_OPEN_GAP_DASHBOARD_V161":
        if "v1.61" not in chain_fixture:
            raise SystemExit("V161 dashboard must use v1.61 rehearsal chain")
        for key, script in (
            ("main_table_signoff_path_fixture", "validate-runbook-main-table-signoff-path-rehearsal.py"),
            ("narrow_defer_inventory_fixture", "validate-r18-narrow-only-defer-inventory.py"),
            ("g4_staging_triple_sync_fixture", "validate-runbook-g4-staging-ticket-triple-sync.py"),
        ):
            fx = link.get(key, "")
            if not fx:
                raise SystemExit(f"V161 dashboard must name {key}")
            run(script, fx)
        # V160 gates still required
        if "v1.60" not in link.get("g4_filled_log_linkage_fixture", ""):
            raise SystemExit("V161 must retain g4_filled_log_linkage v1.60")
        run("validate-runbook-g4-filled-log-sha-linkage.py", link["g4_filled_log_linkage_fixture"])
        run("validate-r21-dashboard-triple-gate-window-registry.py", link["triple_gate_window_registry_fixture"])
        run("validate-r02-matrix-entry-gap-r18-slug-drilldown.py", link["matrix_entry_drilldown_fixture"])
    if link_status == "R21_OPEN_GAP_DASHBOARD_V162":
        if "v1.62" not in chain_fixture:
            raise SystemExit("V162 dashboard must use v1.62 rehearsal chain")
        for key, script in (
            ("main_table_rows1_7_log_fixture", "validate-runbook-main-table-rows1-7-filled-log-rehearsal.py"),
            ("publish_review_narrow_wait_fixture", "validate-r18-publish-review-narrow-wait-decision.py"),
            ("entered_warehouse_crossref_fixture", "validate-runbook-entered-crossref-warehouse-rehearsal.py"),
        ):
            fx = link.get(key, "")
            if not fx:
                raise SystemExit(f"V162 dashboard must name {key}")
            run(script, fx)
    if link_status == "R21_OPEN_GAP_DASHBOARD_V163":
        if "v1.63" not in chain_fixture:
            raise SystemExit("V163 dashboard must use v1.63 rehearsal chain")
        for key, script in (
            ("main_table_rows4_6_shape_fixture", "validate-runbook-main-table-rows4-6-log-shape.py"),
            ("narrow_open_batch_wait_fixture", "validate-r18-narrow-open-batch-wait.py"),
            ("confirmed_warehouse_bundle_fixture", "validate-runbook-confirmed-warehouse-bundle-rehearsal.py"),
        ):
            fx = link.get(key, "")
            if not fx:
                raise SystemExit(f"V163 dashboard must name {key}")
            run(script, fx)
        if "v1.63" not in link.get("main_table_rows1_7_log_fixture", ""):
            raise SystemExit("V163 must use rows1_7 log v1.63")
        run("validate-runbook-main-table-rows1-7-filled-log-rehearsal.py", link["main_table_rows1_7_log_fixture"])
    if link_status == "R21_OPEN_GAP_DASHBOARD_V164":
        if "v1.64" not in chain_fixture:
            raise SystemExit("V164 dashboard must use v1.64 rehearsal chain")
        for key, script in (
            ("main_table_rows4_6_shape_fixture", "validate-runbook-main-table-rows4-6-log-shape.py"),
            ("narrow_open_batch_wait_fixture", "validate-r18-narrow-open-batch-wait.py"),
            ("row5_rollback_shape_fixture", "validate-runbook-main-table-row5-rollback-shape.py"),
            ("teamfill_confirmed_warehouse_window_fixture", "validate-runbook-teamfill-confirmed-warehouse-window.py"),
        ):
            fx = link.get(key, "")
            if not fx:
                raise SystemExit(f"V164 dashboard must name {key}")
            run(script, fx)
        if "v1.64" not in link.get("main_table_rows1_7_log_fixture", ""):
            raise SystemExit("V164 must use rows1_7 log v1.64")
        run("validate-runbook-main-table-rows1-7-filled-log-rehearsal.py", link["main_table_rows1_7_log_fixture"])
        run(
            "validate-runbook-confirmed-warehouse-bundle-rehearsal.py",
            link.get("confirmed_warehouse_bundle_fixture", ""),
        )
    print("OK: R21 open-gap dashboard linkage")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()