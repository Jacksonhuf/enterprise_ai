#!/usr/bin/env python3
"""Validate R09 tier probe summary JSON (v1.17)."""
import json
import sys
from pathlib import Path

REQUIRED_TOP = frozenset({"disclaimer", "generated_at_utc", "tiers", "notes"})
TIER_KEYS = frozenset({"probe", "workers", "elapsed"})
EXPECTED = (
    ("message_load", 32),
    ("message_load", 64),
    ("message_load", 96),
    ("governance_usage", 32),
    ("governance_usage", 64),
    ("governance_usage", 96),
)


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if set(data.keys()) != REQUIRED_TOP:
        raise SystemExit(f"keys must be {sorted(REQUIRED_TOP)}")
    if data["disclaimer"] != "TIER_PROBE_SUMMARY_NOT_SLO":
        raise SystemExit("disclaimer must be TIER_PROBE_SUMMARY_NOT_SLO")
    tiers = data["tiers"]
    if len(tiers) != 6:
        raise SystemExit(f"expected 6 tier rows, got {len(tiers)}")
    seen = []
    for row in tiers:
        if set(row.keys()) != TIER_KEYS:
            raise SystemExit(f"tier keys must be {sorted(TIER_KEYS)}")
        seen.append((row["probe"], row["workers"]))
    if seen != list(EXPECTED):
        raise SystemExit(f"tier order mismatch: {seen}")
    print(f"OK: tier probe summary ({len(tiers)} rows)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()