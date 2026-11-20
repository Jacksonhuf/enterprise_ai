#!/usr/bin/env python3
"""R22 real human UAT signoff shape rehearsal (v1.70)."""
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
    if doc.get("signoff_rehearsal_status") != "R22_REAL_HUMAN_UAT_SIGNOFF_V170":
        raise SystemExit("signoff_rehearsal_status must be R22_REAL_HUMAN_UAT_SIGNOFF_V170")
    if doc.get("real_human_signoff_obtained"):
        raise SystemExit("real_human_signoff_obtained must be false in rehearsal")
    if doc.get("joint_env_ready_for_uat"):
        raise SystemExit("joint_env_ready_for_uat must be false until external hooks connect")
    pilot = DOC / doc.get("pilot_doc", "")
    if not pilot.is_file():
        raise SystemExit("missing pilot_doc")
    roles = doc.get("required_signatory_roles", [])
    if len(roles) < 3:
        raise SystemExit("required_signatory_roles must list at least three roles")
    base = json.loads((FX / doc["base_uat_signoff_shape_fixture"]).read_text())
    for role in roles:
        if "PLACEHOLDER" not in str(base.get(role, "")):
            raise SystemExit(f"base signoff shape {role} must stay PLACEHOLDER")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r02-production-uat-signoff-shape.py"),
            str(FX / doc["base_uat_signoff_shape_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-r22-joint-uat-scenario-rehearsal.py"),
            str(FX / doc["joint_uat_scenario_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    hook = json.loads((FX / doc["joint_environment_hooklist_fixture"]).read_text())
    if hook.get("real_joint_environment_available"):
        raise SystemExit("joint environment must stay unavailable in unsigned rehearsal")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print("OK: R22 real human UAT signoff shape (unsigned rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()