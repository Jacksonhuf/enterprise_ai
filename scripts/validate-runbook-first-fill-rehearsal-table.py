#!/usr/bin/env python3
"""Validate runbook first production fill rehearsal table links (v1.50)."""
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
    doc = json.loads(path.read_text())
    status = doc.get("rehearsal_status")
    if status not in (
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V150",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V152",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V156",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V157",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V158",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V159",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V160",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V161",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V162",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V163",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V164",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V165",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V166",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V167",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V168",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V169",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V170",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V171",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V172",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V173",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V174",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V175",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V176",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V177",
        "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V178",
    ):
        raise SystemExit("rehearsal_status must be V150–V178")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    links = doc.get("links", {})
    run("validate-schema8-production-ledger-first-fill-template.py", links["first_fill_ledger_template"])
    run("validate-production-gate-ticket-crossref-team-fill.py", links["crossref_fixture"])
    run("validate-runbook-section3-defer-rollup-linkage.py", links["defer_linkage_fixture"])
    refs = {r.get("checklist_ref") for r in doc.get("checklist_rows", [])}
    for need in ("C7", "ledger_pending"):
        if need not in refs:
            raise SystemExit(f"checklist_rows must include {need}")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V152":
        for need in ("C8", "log_align"):
            if need not in refs:
                raise SystemExit(f"V152 checklist_rows must include {need}")
        align = links.get("log_alignment_fixture", "")
        if not align or not (FX / align).is_file():
            raise SystemExit("V152 requires log_alignment_fixture file")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V156":
        for need in ("C8", "C9", "log_align"):
            if need not in refs:
                raise SystemExit(f"V156 checklist_rows must include {need}")
        align = links.get("log_alignment_fixture", "")
        if "v1.56" not in align:
            raise SystemExit("V156 requires log_alignment_fixture v1.56")
        if not (FX / align).is_file():
            raise SystemExit("V156 log_alignment_fixture missing")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V157":
        for need in ("C8", "C9", "C10", "C11", "C12", "log_align"):
            if need not in refs:
                raise SystemExit(f"V157 checklist_rows must include {need}")
        align = links.get("log_alignment_fixture", "")
        if "v1.57" not in align:
            raise SystemExit("V157 requires log_alignment_fixture v1.57")
        win = links.get("three_table_window_fixture", "")
        if "v1.57" not in win:
            raise SystemExit("V157 requires three_table_window_fixture v1.57")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V158":
        for need in ("C13", "C14", "C7"):
            if need not in refs:
                raise SystemExit(f"V158 checklist_rows must include {need}")
        defer = links.get("defer_linkage_fixture", "")
        if "v1.58" not in defer:
            raise SystemExit("V158 requires defer_linkage_fixture v1.58")
        joint = links.get("entered_filled_log_joint_fixture", "")
        if "v1.58" not in joint:
            raise SystemExit("V158 requires entered_filled_log_joint_fixture v1.58")
        if "v1.58" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V158 requires log_alignment_fixture v1.58")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V159":
        for need in ("C15", "C16", "C17"):
            if need not in refs:
                raise SystemExit(f"V159 checklist_rows must include {need}")
        if "v1.59" not in links.get("fill_c6_defer_linkage_fixture", ""):
            raise SystemExit("V159 requires fill_c6_defer_linkage_fixture v1.59")
        if "v1.59" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V159 requires log_alignment_fixture v1.59")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V160":
        for need in ("C18", "C19", "C20"):
            if need not in refs:
                raise SystemExit(f"V160 checklist_rows must include {need}")
        if "v1.60" not in links.get("fill_c6_defer_linkage_fixture", ""):
            raise SystemExit("V160 requires fill_c6_defer_linkage_fixture v1.60")
        if "v1.60" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V160 requires log_alignment_fixture v1.60")
        for key in ("g4_filled_log_linkage_fixture", "groups_roles_defer_fixture"):
            if "v1.60" not in links.get(key, ""):
                raise SystemExit(f"V160 requires {key} v1.60")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V161":
        for need in ("C21", "C22", "C23"):
            if need not in refs:
                raise SystemExit(f"V161 checklist_rows must include {need}")
        if "v1.61" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V161 requires log_alignment_fixture v1.61")
        for key in (
            "main_table_signoff_path_fixture",
            "narrow_defer_inventory_fixture",
            "g4_staging_triple_sync_fixture",
        ):
            if "v1.61" not in links.get(key, ""):
                raise SystemExit(f"V161 requires {key} v1.61")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V162":
        for need in ("C24", "C25", "C26"):
            if need not in refs:
                raise SystemExit(f"V162 checklist_rows must include {need}")
        if "v1.62" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V162 requires log_alignment_fixture v1.62")
        for key in (
            "main_table_rows1_7_log_fixture",
            "publish_review_narrow_wait_fixture",
            "entered_warehouse_crossref_fixture",
        ):
            if "v1.62" not in links.get(key, ""):
                raise SystemExit(f"V162 requires {key} v1.62")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V163":
        for need in ("C27", "C28", "C29"):
            if need not in refs:
                raise SystemExit(f"V163 checklist_rows must include {need}")
        if "v1.63" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V163 requires log_alignment_fixture v1.63")
        for key in (
            "main_table_rows1_7_log_fixture",
            "main_table_rows4_6_shape_fixture",
            "narrow_open_batch_wait_fixture",
            "confirmed_warehouse_bundle_fixture",
            "filled_log_shape_fixture",
        ):
            if "v1.63" not in links.get(key, ""):
                raise SystemExit(f"V163 requires {key} v1.63")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V164":
        for need in ("C30", "C31", "C32"):
            if need not in refs:
                raise SystemExit(f"V164 checklist_rows must include {need}")
        if "v1.64" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V164 requires log_alignment_fixture v1.64")
        for key in (
            "main_table_rows1_7_log_fixture",
            "main_table_rows4_6_shape_fixture",
            "narrow_open_batch_wait_fixture",
            "filled_log_shape_fixture",
            "row5_rollback_shape_fixture",
            "teamfill_confirmed_warehouse_window_fixture",
        ):
            if "v1.64" not in links.get(key, ""):
                raise SystemExit(f"V164 requires {key} v1.64")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V165":
        for need in ("C33", "C34", "C35"):
            if need not in refs:
                raise SystemExit(f"V165 checklist_rows must include {need}")
        if "v1.65" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V165 requires log_alignment_fixture v1.65")
        for key in (
            "main_table_signoff_path_fixture",
            "narrow_open_batch_wait_fixture",
            "triple_gate_window_registry_fixture",
            "main_table8_signoff_dashboard_path_fixture",
            "section3_entered_warehouse_intake_fixture",
            "main_table8_dashboard_check_fixture",
        ):
            if "v1.65" not in links.get(key, ""):
                raise SystemExit(f"V165 requires {key} v1.65")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V166":
        for need in ("C36", "C37", "C38", "C39"):
            if need not in refs:
                raise SystemExit(f"V166 checklist_rows must include {need}")
        if "v1.66" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V166 requires log_alignment_fixture v1.66")
        for key in (
            "narrow_open_batch_closeout_fixture",
            "publish_review_narrow_closeout_fixture",
            "main_table_production_signoff_template_fixture",
            "entered_ledger_write_gate_fixture",
        ):
            if "v1.66" not in links.get(key, ""):
                raise SystemExit(f"V166 requires {key} v1.66")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V167":
        for need in ("C40", "C41", "C42"):
            if need not in refs:
                raise SystemExit(f"V167 checklist_rows must include {need}")
        if "v1.67" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V167 requires log_alignment_fixture v1.67")
        for key in (
            "figma_pixel_green_uat_fixture",
            "sre_platform_dual_signoff_fixture",
            "dsn_dry_run_ledger_fixture",
        ):
            if "v1.67" not in links.get(key, ""):
                raise SystemExit(f"V167 requires {key} v1.67")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V168":
        for need in ("C43", "C44", "C45"):
            if need not in refs:
                raise SystemExit(f"V168 checklist_rows must include {need}")
        if "v1.68" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V168 requires log_alignment_fixture v1.68")
        for key in (
            "global_gap_item_closeout_fixture",
            "r22_joint_uat_scenario_fixture",
            "production_apply_preflight_fixture",
        ):
            if "v1.68" not in links.get(key, ""):
                raise SystemExit(f"V168 requires {key} v1.68")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V169":
        for need in ("C46", "C47", "C48"):
            if need not in refs:
                raise SystemExit(f"V169 checklist_rows must include {need}")
        if "v1.69" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V169 requires log_alignment_fixture v1.69")
        for key in (
            "s5_release_ops_checklist_fixture",
            "real_joint_environment_hooklist_fixture",
            "rehearsal_only_gate_fixture",
        ):
            if "v1.69" not in links.get(key, ""):
                raise SystemExit(f"V169 requires {key} v1.69")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V170":
        for need in ("C49", "C50"):
            if need not in refs:
                raise SystemExit(f"V170 checklist_rows must include {need}")
        if "v1.70" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V170 requires log_alignment_fixture v1.70")
        for key in (
            "real_human_uat_signoff_fixture",
            "unsigned_uat_apply_block_fixture",
        ):
            if "v1.70" not in links.get(key, ""):
                raise SystemExit(f"V170 requires {key} v1.70")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V171":
        for need in ("C51", "C52"):
            if need not in refs:
                raise SystemExit(f"V171 checklist_rows must include {need}")
        if "v1.71" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V171 requires log_alignment_fixture v1.71")
        for key in (
            "joint_env_connected_shape_fixture",
            "joint_env_connected_unsigned_gate_fixture",
        ):
            if "v1.71" not in links.get(key, ""):
                raise SystemExit(f"V171 requires {key} v1.71")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V172":
        for need in ("C53", "C54"):
            if need not in refs:
                raise SystemExit(f"V172 checklist_rows must include {need}")
        if "v1.72" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V172 requires log_alignment_fixture v1.72")
        for key in (
            "uat_signoff_dryrun_ledger_fixture",
            "joint_env_shape_to_live_gate_fixture",
        ):
            if "v1.72" not in links.get(key, ""):
                raise SystemExit(f"V172 requires {key} v1.72")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V173":
        for need in ("C55", "C56"):
            if need not in refs:
                raise SystemExit(f"V173 checklist_rows must include {need}")
        if "v1.73" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V173 requires log_alignment_fixture v1.73")
        for key in (
            "r20_pressure_sse_measurement_shape_fixture",
            "r20_s5_capacity_gate_fixture",
        ):
            if "v1.73" not in links.get(key, ""):
                raise SystemExit(f"V173 requires {key} v1.73")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V174":
        for need in ("C57", "C58"):
            if need not in refs:
                raise SystemExit(f"V174 checklist_rows must include {need}")
        if "v1.74" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V174 requires log_alignment_fixture v1.74")
        for key in (
            "r21_ops_observability_ledger_fixture",
            "r21_backup_rpo_rto_gate_fixture",
        ):
            if "v1.74" not in links.get(key, ""):
                raise SystemExit(f"V174 requires {key} v1.74")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V175":
        for need in ("C59", "C60"):
            if need not in refs:
                raise SystemExit(f"V175 checklist_rows must include {need}")
        if "v1.75" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V175 requires log_alignment_fixture v1.75")
        for key in (
            "r21_alert_dryrun_ledger_fixture",
            "r21_oncall_handoff_gate_fixture",
        ):
            if "v1.75" not in links.get(key, ""):
                raise SystemExit(f"V175 requires {key} v1.75")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V176":
        for need in ("C61", "C62"):
            if need not in refs:
                raise SystemExit(f"V176 checklist_rows must include {need}")
        if "v1.76" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V176 requires log_alignment_fixture v1.76")
        for key in (
            "r20_formal_slo_target_ledger_fixture",
            "r20_slo_evidence_rehearsal_gate_fixture",
        ):
            if "v1.76" not in links.get(key, ""):
                raise SystemExit(f"V176 requires {key} v1.76")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V177":
        for need in ("C63", "C64"):
            if need not in refs:
                raise SystemExit(f"V177 checklist_rows must include {need}")
        if "v1.77" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V177 requires log_alignment_fixture v1.77")
        for key in (
            "r21_backup_slo_recheck_ledger_fixture",
            "r21_backup_slo_joint_gate_fixture",
        ):
            if "v1.77" not in links.get(key, ""):
                raise SystemExit(f"V177 requires {key} v1.77")
    if status == "RUNBOOK_FIRST_FILL_REHEARSAL_TABLE_V178":
        for need in ("C65", "C66"):
            if need not in refs:
                raise SystemExit(f"V178 checklist_rows must include {need}")
        if "v1.78" not in links.get("log_alignment_fixture", ""):
            raise SystemExit("V178 requires log_alignment_fixture v1.78")
        for key in (
            "r22_joint_uat_slo_crossref_ledger_fixture",
            "r22_joint_uat_slo_signoff_gate_fixture",
        ):
            if "v1.78" not in links.get(key, ""):
                raise SystemExit(f"V178 requires {key} v1.78")
    for row in doc.get("checklist_rows", []):
        val = str(row.get("rehearsal_value", ""))
        if "PLACEHOLDER" in val.upper():
            raise SystemExit("rehearsal_value must not use PLACEHOLDER token")
    print("OK: runbook first-fill rehearsal table")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()