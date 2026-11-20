#!/usr/bin/env python3
"""SRE + platform dual signoff shape rehearsal (v1.67)."""
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
    if doc.get("shape_status") != "RUNBOOK_SRE_PLATFORM_DUAL_SIGNOFF_V167":
        raise SystemExit("shape_status must be RUNBOOK_SRE_PLATFORM_DUAL_SIGNOFF_V167")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r21-slo-signoff-record.py"), str(FX / doc["slo_signoff_pending_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r21-slo-signoff-record.py"), str(FX / doc["slo_signoff_shape_example_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-main-table-production-signoff-template.py"),
            str(FX / doc["main_table_signoff_template_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    log = json.loads((FX / doc["filled_log_fixture"]).read_text())
    for key in (doc.get("required_platform_field"), doc.get("required_sre_field")):
        val = str(log.get(key, ""))
        if "PENDING" not in val and "PLACEHOLDER" not in val:
            raise SystemExit(f"log {key} must stay rehearsal pending")
    example = json.loads((FX / doc["slo_signoff_shape_example_fixture"]).read_text())
    for owner in ("platform_engineering_owner", "sre_owner"):
        if "PLACEHOLDER" not in str(example.get(owner, "")):
            raise SystemExit("shape example owners must be PLACEHOLDER")
    if not doc.get("dual_signoff_still_placeholder"):
        raise SystemExit("dual_signoff_still_placeholder must be true")
    print("OK: SRE/platform dual signoff shape (rehearsal)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()