#!/usr/bin/env python3
"""Validate Figma pending fields have an engineering closeout plan (v1.53)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    plan = json.loads(path.read_text())
    status = plan.get("plan_status")
    if status not in ("R02_FIGMA_PENDING_CLOSEOUT_PLAN_V153", "R02_FIGMA_PENDING_CLOSEOUT_PLAN_V154"):
        raise SystemExit("plan_status must be V153 or V154")
    figma_path = FX / plan["figma_checklist_fixture"]
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-figma-field-checklist.py"), str(figma_path)],
        check=True,
        cwd=ROOT,
    )
    figma = json.loads(figma_path.read_text())
    pending: list[tuple[str, str]] = []
    for entry in figma.get("entries", []):
        did = entry["design_id"]
        for f in entry.get("fields", []):
            if f.get("status") == "pending":
                pending.append((did, f["id"]))
    planned = {(t["design_id"], t["field_id"]) for t in plan.get("pending_targets", [])}
    if set(pending) != planned:
        missing = set(pending) - planned
        extra = planned - set(pending)
        if missing:
            raise SystemExit(f"pending fields without plan: {sorted(missing)}")
        if extra:
            raise SystemExit(f"plan entries not pending: {sorted(extra)}")
    for t in plan.get("pending_targets", []):
        if not t.get("planned_evidence") or not t.get("target_round"):
            raise SystemExit("each target needs planned_evidence and target_round")
    matched_by_design: dict[str, set[str]] = {}
    for entry in figma.get("entries", []):
        matched_by_design[entry["design_id"]] = {
            f["id"] for f in entry.get("fields", []) if f.get("status") == "matched"
        }
    for item in plan.get("fulfilled_in_v154", []):
        did, fid = item["design_id"], item["field_id"]
        if fid not in matched_by_design.get(did, set()):
            raise SystemExit(f"fulfilled {did}/{fid} must be matched in checklist")
    print(f"OK: Figma pending closeout plan ({len(pending)} pending)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()