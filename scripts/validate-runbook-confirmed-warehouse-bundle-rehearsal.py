#!/usr/bin/env python3
"""CONFIRMED bundle with warehouse ticket format rehearsal (v1.63)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("rehearsal_status") != "CONFIRMED_WAREHOUSE_BUNDLE_V163":
        raise SystemExit("rehearsal_status must be CONFIRMED_WAREHOUSE_BUNDLE_V163")
    if not doc.get("not_production"):
        raise SystemExit("not_production must be true")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-production-gate-crossref-bundle-joint.py"),
            str(FX / doc["confirmed_joint_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-runbook-entered-crossref-warehouse-rehearsal.py"), str(FX / doc["entered_warehouse_fixture"])],
        check=True,
        cwd=ROOT,
    )
    bundle = json.loads((FX / doc["bundle_fixture"]).read_text())
    if bundle.get("bundle_status") != "CONFIRMED_SHAPE_ONLY":
        raise SystemExit("bundle must be CONFIRMED_SHAPE_ONLY")
    xref = json.loads((FX / doc["crossref_fixture"]).read_text())
    if "REHEARSAL" in xref.get("apply_change_ticket_ref", "").upper():
        raise SystemExit("warehouse crossref must use non-REHEARSAL ticket shape")
    for field in ("confirmed_by_platform", "confirmed_by_sre"):
        if not str(bundle.get(field, "")).startswith("PLACEHOLDER"):
            raise SystemExit(f"{field} must be PLACEHOLDER in rehearsal")
    print("OK: CONFIRMED warehouse bundle rehearsal shape")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()