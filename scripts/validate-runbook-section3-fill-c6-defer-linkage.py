#!/usr/bin/env python3
"""Link §3 fill checklist C6 with defer rollup counts (v1.59)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def run(script: str, fixture: str | None = None) -> None:
    cmd = [PY, str(ROOT / "scripts" / script)]
    if fixture:
        cmd.append(str(FX / fixture))
    subprocess.run(cmd, check=True, cwd=ROOT)


def main() -> None:
    path = Path(sys.argv[1])
    link = json.loads(path.read_text())
    link_status = link.get("linkage_status")
    if link_status not in (
        "RUNBOOK_SECTION3_FILL_C6_DEFER_V159",
        "RUNBOOK_SECTION3_FILL_C6_DEFER_V160",
    ):
        raise SystemExit("linkage_status must be V159 or V160")
    run("validate-runbook-section3-fill-checklist-rehearsal.py", link["fill_checklist_fixture"])
    run("validate-r18-headroom-defer-rollup.py", link["defer_rollup_fixture"])
    run("validate-runbook-section3-defer-rollup-linkage.py", link["defer_linkage_fixture"])
    if link.get("triple_gate_script") == "validate-runbook-triple-gate-rehearsal.py":
        run("validate-runbook-triple-gate-rehearsal.py")
    rollup = json.loads((FX / link["defer_rollup_fixture"]).read_text())
    if len(rollup.get("defer_records", [])) != int(link.get("expected_defer_record_count", 0)):
        raise SystemExit("defer record count mismatch")
    slugs = {r["slug"] for r in rollup["defer_records"]}
    for need in link.get("c6_defer_ratio_keys", []):
        if need not in slugs:
            raise SystemExit(f"C6 defer linkage missing slug {need}")
    checklist = json.loads((FX / link["fill_checklist_fixture"]).read_text())
    if not checklist.get("rehearsal_checkmarks", {}).get("C15", {}).get("checked"):
        raise SystemExit("fill checklist must mark C15 for this linkage")
    if link_status == "RUNBOOK_SECTION3_FILL_C6_DEFER_V160":
        mt8 = link.get("main_table8_check_fixture", "")
        if not mt8:
            raise SystemExit("V160 requires main_table8_check_fixture")
        run("validate-runbook-main-table8-dashboard-window-check.py", mt8)
    print(f"OK: §3 fill C6↔defer ({len(slugs)} defer slugs)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()