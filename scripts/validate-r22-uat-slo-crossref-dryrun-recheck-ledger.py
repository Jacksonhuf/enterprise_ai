#!/usr/bin/env python3
"""R22 UAT dry-run ↔ UAT/SLO crossref same-window recheck (v1.79)."""
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
    if doc.get("ledger_status") != "R22_UAT_SLO_CROSSREF_DRYRUN_RECHECK_V179":
        raise SystemExit("ledger_status must be R22_UAT_SLO_CROSSREF_DRYRUN_RECHECK_V179")
    if doc.get("joint_recheck_passed"):
        raise SystemExit("joint_recheck_passed must be false in rehearsal")
    for name in doc.get("pilot_docs", []):
        if not (DOC / name).is_file():
            raise SystemExit(f"missing pilot doc {name}")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-real-human-uat-signoff-dryrun-ledger.py"),
            str(FX / doc["uat_signoff_dryrun_ledger_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-joint-uat-slo-signoff-crossref-ledger.py"),
            str(FX / doc["joint_uat_slo_crossref_ledger_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    dryrun = json.loads((FX / doc["uat_signoff_dryrun_ledger_fixture"]).read_text())
    crossref = json.loads((FX / doc["joint_uat_slo_crossref_ledger_fixture"]).read_text())
    if dryrun.get("real_human_signoff_obtained"):
        raise SystemExit("dryrun ledger real_human_signoff_obtained must be false")
    if crossref.get("slo_signoff_crossref_ready"):
        raise SystemExit("crossref slo_signoff_crossref_ready must be false")
    dry_personas = {e["persona"] for e in dryrun.get("ledger_entries", [])}
    uat = json.loads((FX / crossref["joint_uat_scenario_fixture"]).read_text())
    scenario_personas = {s["persona"] for s in uat.get("scenarios", [])}
    if dry_personas != scenario_personas:
        raise SystemExit("dry-run personas must match joint UAT scenarios")
    items = doc.get("recheck_items", [])
    if len(items) != int(doc.get("expected_recheck_count", 0)):
        raise SystemExit("recheck_items count mismatch")
    for item in items:
        if item.get("recheck_executed"):
            raise SystemExit(f"{item.get('recheck_id')} recheck_executed must be false")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r21-slo-signoff-record.py"),
            str(FX / "r21-slo-signoff-record-pending.json"),
        ],
        check=True,
        cwd=ROOT,
    )
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: R22 UAT/SLO dry-run recheck ledger ({len(items)} items)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()