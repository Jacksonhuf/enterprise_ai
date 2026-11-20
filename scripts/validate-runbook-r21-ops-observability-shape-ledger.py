#!/usr/bin/env python3
"""R21 ops observability shape ledger (backup/RPO/RTO, v1.74)."""
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
    if doc.get("ledger_status") != "R21_OPS_OBSERVABILITY_SHAPE_LEDGER_V174":
        raise SystemExit("ledger_status must be R21_OPS_OBSERVABILITY_SHAPE_LEDGER_V174")
    for name in doc.get("pilot_docs", []):
        if not (DOC / name).is_file():
            raise SystemExit(f"missing pilot doc {name}")
    items = doc.get("ops_items", [])
    if len(items) != int(doc.get("expected_item_count", 0)):
        raise SystemExit("ops_items count mismatch")
    for item in items:
        if item.get("restore_verified") or item.get("targets_adopted") or item.get("observability_gate_met"):
            raise SystemExit(f"{item.get('item_id')} gates must stay false in rehearsal")
        item_id = item.get("item_id", "")
        if item_id == "mysql-backup-restore":
            if "PLACEHOLDER" not in str(item.get("backup_snapshot_shape_ref", "")):
                raise SystemExit("backup_snapshot_shape_ref must be PLACEHOLDER")
        if item_id == "rpo-rto-targets":
            for key in ("rpo_hours_shape", "rto_hours_shape"):
                if "PLACEHOLDER" not in str(item.get(key, "")):
                    raise SystemExit(f"{key} must be PLACEHOLDER")
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
        [
            PY,
            str(ROOT / "scripts/validate-runbook-sre-platform-dual-signoff-shape.py"),
            str(FX / doc["dual_signoff_shape_fixture"]),
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
    backup = str(log.get("rollback_approval_shape", {}).get("backup_path_or_snapshot_sha256", ""))
    if "PLACEHOLDER" not in backup and "REHEARSAL" not in backup:
        raise SystemExit("filled log backup shape must stay rehearsal placeholder")
    if not doc.get("production_apply_forbidden"):
        raise SystemExit("production_apply_forbidden must be true")
    print(f"OK: R21 ops observability shape ledger ({len(items)} items)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()