#!/usr/bin/env python3
"""Validate schema8 rehearsal apply log JSON (v1.06); production records use the same shape."""
import json
import re
import sys
from pathlib import Path

REQUIRED_STEPS = (
    "audit",
    "applications",
    "grants",
    "sessions",
    "settings",
    "exceptions",
    "usage",
)
SHA = re.compile(r"^[0-9a-f]{64}$")


def validate(record: dict) -> None:
    if record.get("disclaimer") != "REHEARSAL_ONLY — 非生产库，勿写入生产台账":
        if not record.get("production_signoff_ticket"):
            raise SystemExit("production record requires production_signoff_ticket or REHEARSAL_ONLY disclaimer")
    for key in (
        "executed_at_cst",
        "dsn_database_host",
        "change_ticket",
        "migrate_plan_sha256",
        "apply_exit_codes",
        "dual_instance_ready",
    ):
        if key not in record:
            raise SystemExit(f"missing field: {key}")
    plans = record["migrate_plan_sha256"]
    if set(plans.keys()) != set(REQUIRED_STEPS):
        raise SystemExit(f"migrate_plan_sha256 keys must be {REQUIRED_STEPS}")
    for step, digest in plans.items():
        if not SHA.fullmatch(str(digest)):
            raise SystemExit(f"invalid sha256 for {step}")
    codes = record["apply_exit_codes"]
    if not isinstance(codes, list) or len(codes) != 7 or any(c != 0 for c in codes):
        raise SystemExit("apply_exit_codes must be seven zeros")
    if record["dual_instance_ready"] is not True:
        raise SystemExit("dual_instance_ready must be true for acceptance")
    if record["change_ticket"].startswith("REHEARSAL-"):
        print("OK: rehearsal record", record["change_ticket"])
    else:
        print("OK: production-shaped record", record["change_ticket"])


def main() -> None:
    path = Path(sys.argv[1])
    validate(json.loads(path.read_text()))
    print(f"Validated {path}")


if __name__ == "__main__":
    main()