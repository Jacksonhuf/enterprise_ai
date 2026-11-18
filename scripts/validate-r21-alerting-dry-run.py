#!/usr/bin/env python3
"""Validate R21 alerting dry-run rules JSON (v1.07); not deployed to production."""
import json
import sys
from pathlib import Path

REQUIRED = frozenset({"name", "metric", "threshold_seconds", "pilot_observed_seconds", "source_test"})


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if data.get("disclaimer") != "ALERTING_DRY_RUN_ONLY":
        raise SystemExit("disclaimer must be ALERTING_DRY_RUN_ONLY")
    rules = data.get("rules")
    if not isinstance(rules, list) or not rules:
        raise SystemExit("rules must be non-empty list")
    for rule in rules:
        if set(rule.keys()) != REQUIRED:
            raise SystemExit(f"rule keys must be {sorted(REQUIRED)}: {rule.get('name')}")
        if rule["threshold_seconds"] < rule["pilot_observed_seconds"]:
            raise SystemExit(f"threshold below pilot for {rule['name']}")
        if rule["threshold_seconds"] > rule["pilot_observed_seconds"] * 4:
            raise SystemExit(f"threshold unreasonably high for {rule['name']}")
    print(f"OK: {len(rules)} dry-run alerting rules validated")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()