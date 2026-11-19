#!/usr/bin/env python3
"""Validate production exporter review checklist JSON (v1.24)."""
import json
import sys
from pathlib import Path

REQUIRED = frozenset(
    {
        "checklist_status",
        "items_passed",
        "reviewed_by_platform",
        "reviewed_by_sre",
        "reviewed_at_cst",
        "linked_exporter_review_doc",
        "notes",
    }
)


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if set(data.keys()) != REQUIRED:
        raise SystemExit(f"keys must be {sorted(REQUIRED)}")
    if data["items_passed"] != 6:
        raise SystemExit("items_passed must be 6")
    if data["checklist_status"] == "APPROVED":
        for k in ("reviewed_by_platform", "reviewed_by_sre", "reviewed_at_cst"):
            if not data[k] or "PLACEHOLDER" in data[k]:
                raise SystemExit(f"APPROVED requires real {k}")
    elif data["checklist_status"] not in (
        "REVIEW_SHAPE_ONLY",
        "APPROVED_SHAPE_ONLY",
        "PENDING_REVIEW",
    ):
        raise SystemExit("invalid checklist_status")
    print(f"OK: exporter review ({data['checklist_status']})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()