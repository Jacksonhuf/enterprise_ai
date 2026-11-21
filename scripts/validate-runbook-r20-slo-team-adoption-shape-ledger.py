#!/usr/bin/env python3
"""R20 SLO team adoption shape ledger (v1.80, mock)."""
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
    if doc.get("ledger_status") != "R20_SLO_TEAM_ADOPTION_SHAPE_V180":
        raise SystemExit("ledger_status must be R20_SLO_TEAM_ADOPTION_SHAPE_V180")
    if doc.get("team_slo_adopted"):
        raise SystemExit("team_slo_adopted must be false in rehearsal")
    for name in doc.get("pilot_docs", []):
        if not (DOC / name).is_file():
            raise SystemExit(f"missing pilot doc {name}")
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
            str(ROOT / "scripts/validate-r22-uat-slo-crossref-dryrun-gate-rehearsal.py"),
            str(FX / doc["uat_slo_dryrun_gate_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    formal = json.loads((FX / doc["formal_slo_target_ledger_fixture"]).read_text())
    target_ids = {t["target_id"] for t in formal.get("slo_targets", [])}
    rows = doc.get("adoption_rows", [])
    if len(rows) != int(doc.get("expected_adoption_count", 0)):
        raise SystemExit("adoption_rows count mismatch")
    for row in rows:
        if row.get("team_review_recorded"):
            raise SystemExit(f"{row.get('target_id')} team_review_recorded must be false")
        if "PLACEHOLDER" not in str(row.get("proposed_value_shape", "")):
            raise SystemExit("proposed_value_shape must use PLACEHOLDER")
        if row.get("target_id") not in target_ids:
            raise SystemExit(f"unknown target_id {row.get('target_id')}")
    dryrun = json.loads((FX / doc["uat_slo_dryrun_gate_fixture"]).read_text())
    if dryrun.get("release_window_id") != doc.get("release_window_id"):
        raise SystemExit("release_window_id must match dry-run gate")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r21-slo-signoff-record.py"),
            str(FX / doc["slo_signoff_pending_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: R20 SLO team adoption shape ({len(rows)} rows, mock)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()