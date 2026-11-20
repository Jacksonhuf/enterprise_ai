#!/usr/bin/env python3
"""R20 pressure/SSE fault measurement shape (v1.73, mock only)."""
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
    if doc.get("measurement_status") != "R20_PRESSURE_SSE_FAULT_MEASUREMENT_SHAPE_V173":
        raise SystemExit("measurement_status must be R20_PRESSURE_SSE_FAULT_MEASUREMENT_SHAPE_V173")
    if doc.get("formal_load_test_completed"):
        raise SystemExit("formal_load_test_completed must be false in rehearsal")
    pilot = DOC / doc.get("pilot_doc", "")
    if not pilot.is_file():
        raise SystemExit("missing pilot_doc")
    probes = doc.get("measurement_probes", [])
    if len(probes) != int(doc.get("expected_probe_count", 0)):
        raise SystemExit("probe count mismatch")
    go_tests = ""
    if API.is_dir():
        for go_file in API.glob("*.go"):
            go_tests += go_file.read_text()
    elif API.is_file():
        go_tests = API.read_text()
    for probe in probes:
        if probe.get("slo_gate_met"):
            raise SystemExit(f"probe {probe.get('probe_id')} slo_gate_met must be false")
        if "PLACEHOLDER" not in str(probe.get("recorded_value_shape", "")):
            raise SystemExit("recorded_value_shape must use PLACEHOLDER")
        test_name = probe.get("harness_test", "")
        if test_name and test_name not in go_tests:
            raise SystemExit(f"harness_test {test_name} not found in schema8 R20 tests")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-s5-release-ops-checklist-rehearsal.py"),
            str(FX / doc["s5_release_ops_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    s5 = json.loads((FX / doc["s5_release_ops_fixture"]).read_text())
    r20_item = next((i for i in s5.get("ops_items", []) if i.get("item_id") == "R20-capacity-fault"), None)
    if not r20_item or r20_item.get("exit_gate_met"):
        raise SystemExit("S5 R20-capacity-fault exit_gate_met must be false")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / doc["filled_log_fixture"])],
        check=True,
        cwd=ROOT,
    )
    log = json.loads((FX / doc["filled_log_fixture"]).read_text())
    if not log.get("dual_instance_ready"):
        raise SystemExit("filled log dual_instance_ready must be true for R20 shape")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: R20 pressure/SSE measurement shape ({len(probes)} probes, mock)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()