#!/usr/bin/env python3
"""R21 alert dry-run shape ledger (v1.75)."""
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
    if doc.get("ledger_status") != "R21_ALERT_DRYRUN_SHAPE_LEDGER_V175":
        raise SystemExit("ledger_status must be R21_ALERT_DRYRUN_SHAPE_LEDGER_V175")
    for name in doc.get("pilot_docs", []):
        if not (DOC / name).is_file():
            raise SystemExit(f"missing pilot doc {name}")
    items = doc.get("alert_dryrun_items", [])
    if len(items) != int(doc.get("expected_item_count", 0)):
        raise SystemExit("alert_dryrun_items count mismatch")
    for item in items:
        if item.get("dryrun_fired"):
            raise SystemExit(f"{item.get('alert_id')} dryrun_fired must stay false")
        for key in ("pager_route_shape", "ack_owner"):
            if "PLACEHOLDER" not in str(item.get(key, "")):
                raise SystemExit(f"{item.get('alert_id')} {key} must be PLACEHOLDER")
    subprocess.run(
        [
            PY,
            str(ROOT / "scripts/validate-runbook-r21-ops-observability-shape-ledger.py"),
            str(FX / doc["ops_observability_ledger_fixture"]),
        ],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-rehearsal-log-record.py"), str(FX / doc["filled_log_fixture"])],
        check=True,
        cwd=ROOT,
    )
    log = json.loads((FX / doc["filled_log_fixture"]).read_text())
    slo = str(log.get("r21_slo_signoff_status", ""))
    if "PENDING" not in slo and "PLACEHOLDER" not in slo:
        raise SystemExit("r21_slo_signoff_status must stay pending in rehearsal")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: R21 alert dry-run shape ledger ({len(items)} items)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()