#!/usr/bin/env python3
"""R20 formal SLO target shape ledger (v1.76, mock)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
DOC = ROOT / "docs/development"
API = ROOT / "backend/platform/internal/platform/api"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("ledger_status") != "R20_FORMAL_SLO_TARGET_SHAPE_V176":
        raise SystemExit("ledger_status must be R20_FORMAL_SLO_TARGET_SHAPE_V176")
    if doc.get("formal_slo_adopted"):
        raise SystemExit("formal_slo_adopted must be false in rehearsal")
    for name in doc.get("pilot_docs", []):
        if not (DOC / name).is_file():
            raise SystemExit(f"missing pilot doc {name}")
    measurement = json.loads((FX / doc["measurement_shape_fixture"]).read_text())
    probe_ids = {p["probe_id"] for p in measurement.get("measurement_probes", [])}
    targets = doc.get("slo_targets", [])
    if len(targets) != int(doc.get("expected_target_count", 0)):
        raise SystemExit("slo_targets count mismatch")
    go_tests = ""
    if API.is_dir():
        for go_file in API.glob("*.go"):
            go_tests += go_file.read_text()
    for target in targets:
        if target.get("target_adopted"):
            raise SystemExit(f"{target.get('target_id')} target_adopted must be false")
        if "PLACEHOLDER" not in str(target.get("target_value_shape", "")):
            raise SystemExit("target_value_shape must use PLACEHOLDER")
        if target.get("linked_probe_id") not in probe_ids:
            raise SystemExit(f"unknown linked_probe_id {target.get('linked_probe_id')}")
        test_name = target.get("evidence_harness_test", "")
        if test_name and test_name not in go_tests:
            raise SystemExit(f"evidence_harness_test {test_name} not found")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r20-pressure-sse-fault-measurement-shape.py"),
            str(FX / doc["measurement_shape_fixture"]),
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
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / doc["filled_log_fixture"])],
        check=True,
        cwd=ROOT,
    )
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: R20 formal SLO target shape ({len(targets)} targets, mock)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()