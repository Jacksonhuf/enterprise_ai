#!/usr/bin/env python3
"""R22 joint UAT ↔ R21/R20 SLO signoff crossref ledger (v1.78)."""
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
    if doc.get("ledger_status") != "R22_JOINT_UAT_SLO_SIGNOFF_CROSSREF_V178":
        raise SystemExit("ledger_status must be R22_JOINT_UAT_SLO_SIGNOFF_CROSSREF_V178")
    if doc.get("slo_signoff_crossref_ready"):
        raise SystemExit("slo_signoff_crossref_ready must be false in rehearsal")
    for name in doc.get("pilot_docs", []):
        if not (DOC / name).is_file():
            raise SystemExit(f"missing pilot doc {name}")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-joint-uat-scenario-rehearsal.py"),
            str(FX / doc["joint_uat_scenario_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r20-formal-slo-target-shape-ledger.py"),
            str(FX / doc["formal_slo_target_ledger_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r21-slo-signoff-record.py"),
            str(FX / doc["slo_signoff_pending_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    uat = json.loads((FX / doc["joint_uat_scenario_fixture"]).read_text())
    scenario_ids = {s["scenario_id"] for s in uat.get("scenarios", [])}
    slo_doc = json.loads((FX / doc["formal_slo_target_ledger_fixture"]).read_text())
    target_ids = {t["target_id"] for t in slo_doc.get("slo_targets", [])}
    rows = doc.get("crossref_rows", [])
    if len(rows) != int(doc.get("expected_crossref_count", 0)):
        raise SystemExit("crossref_rows count mismatch")
    for row in rows:
        if "PLACEHOLDER" not in str(row.get("uat_evidence_shape", "")):
            raise SystemExit("uat_evidence_shape must use PLACEHOLDER")
        if row.get("scenario_id") not in scenario_ids:
            raise SystemExit(f"unknown scenario_id {row.get('scenario_id')}")
        if row.get("linked_slo_target_id") not in target_ids:
            raise SystemExit(f"unknown linked_slo_target_id {row.get('linked_slo_target_id')}")
    signoff = json.loads((FX / doc["slo_signoff_pending_fixture"]).read_text())
    if signoff.get("record_status") != "PENDING_TEAM_SIGN":
        raise SystemExit("slo signoff must stay PENDING_TEAM_SIGN")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: R22 joint UAT SLO crossref ledger ({len(rows)} rows, mock)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()