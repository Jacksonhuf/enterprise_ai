#!/usr/bin/env python3
"""Register triple-gate rehearsal on same window as open-gap dashboard (v1.59)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    reg = json.loads(path.read_text())
    if reg.get("registry_status") != "R21_DASHBOARD_TRIPLE_GATE_WINDOW_V159":
        raise SystemExit("registry_status must be R21_DASHBOARD_TRIPLE_GATE_WINDOW_V159")
    dash_name = reg.get("dashboard_linkage_fixture", "")
    dash = json.loads((FX / dash_name).read_text())
    if dash.get("linkage_status") != "R21_OPEN_GAP_DASHBOARD_V159":
        raise SystemExit("dashboard must be V159")
    if dash.get("triple_gate_window_registry_fixture") != path.name:
        raise SystemExit("dashboard must point to this registry")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-production-gate-three-table-window.py"), str(FX / reg["three_table_window_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run([PY, str(ROOT / "scripts" / reg["triple_gate_script"])], check=True, cwd=ROOT)
    win = json.loads((FX / reg["three_table_window_fixture"]).read_text())
    if win.get("release_window_id") != reg.get("release_window_id"):
        raise SystemExit("release_window_id mismatch")
    if dash.get("three_table_window_fixture") != reg["three_table_window_fixture"]:
        raise SystemExit("dashboard three_table_window must match registry")
    if dash.get("defer_rollup_linkage_fixture") != reg.get("defer_linkage_fixture"):
        raise SystemExit("defer linkage must match between dashboard and registry")
    print(f"OK: dashboard↔triple-gate window ({reg['release_window_id']})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()