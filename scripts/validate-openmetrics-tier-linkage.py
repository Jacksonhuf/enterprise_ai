#!/usr/bin/env python3
"""Verify OpenMetrics tier metrics match tier probe summary (v1.19)."""
import json
import re
import sys
from pathlib import Path

TIER_METRIC = {
    ("message_load", 64): "cognara_fg_msg_load_tier64_seconds",
    ("message_load", 96): "cognara_fg_msg_load_tier96_seconds",
    ("governance_usage", 64): "cognara_fg_gov_usage_tier64_seconds",
    ("governance_usage", 96): "cognara_fg_gov_usage_tier96_seconds",
}


def to_seconds(elapsed: str) -> float:
    if elapsed.endswith("ms"):
        return float(elapsed[:-2]) / 1000.0
    if elapsed.endswith("s"):
        return float(elapsed[:-1])
    raise ValueError(elapsed)


def main() -> None:
    tier_path = Path(sys.argv[1])
    om_path = Path(sys.argv[2])
    tiers = json.loads(tier_path.read_text())["tiers"]
    text = om_path.read_text()
    got = {
        m.group(1): float(m.group(2))
        for m in re.finditer(r"^(cognara_fg_\w+)\s+([\d.eE+-]+)\s*$", text, re.MULTILINE)
    }
    for row in tiers:
        key = (row["probe"], row["workers"])
        name = TIER_METRIC.get(key)
        if not name:
            continue
        if name not in got:
            raise SystemExit(f"missing metric {name}")
        exp = to_seconds(row["elapsed"])
        if abs(got[name] - exp) > 0.001:
            raise SystemExit(f"{name}: want {exp} got {got[name]}")
    print("OK: openmetrics tier linkage")
    print(f"Validated {om_path} against {tier_path}")


if __name__ == "__main__":
    main()