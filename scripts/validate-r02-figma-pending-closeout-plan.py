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
    if plan.get("plan_status") != "R02_FIGMA_PENDING_CLOSEOUT_PLAN_V153":
        raise SystemExit("plan_status must be R02_FIGMA_PENDING_CLOSEOUT_PLAN_V153")
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
    print(f"OK: Figma pending closeout plan ({len(pending)} pending)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()