#!/usr/bin/env python3
"""Overlay tier probe elapsed onto OpenMetrics draft (v1.19)."""
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DRAFT = ROOT / "docs/development/fixtures/openmetrics-exporter-draft.txt"

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
    raise ValueError(f"unsupported elapsed: {elapsed}")


def main() -> None:
    tier_path = Path(sys.argv[1])
    out_path = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("/tmp/openmetrics-from-tier.txt")
    tiers = json.loads(tier_path.read_text())["tiers"]
    overrides = {}
    for row in tiers:
        key = (row["probe"], row["workers"])
        name = TIER_METRIC.get(key)
        if name:
            overrides[name] = to_seconds(row["elapsed"])
    text = DRAFT.read_text()
    if "OPENMETRICS_EXPORTER_DRAFT_ONLY" not in text:
        raise SystemExit("draft missing banner")
    lines = []
    for line in text.splitlines():
        m = re.match(r"^(cognara_fg_\w+)\s+([\d.]+)\s*$", line)
        if m and m.group(1) in overrides:
            lines.append(f"{m.group(1)} {overrides[m.group(1)]:.6g}")
        else:
            lines.append(line)
    header = "# OPENMETRICS_TIER_LINKED — values from tier probe summary\n"
    out_path.write_text(header + "\n".join(lines) + "\n")
    print(f"Wrote {out_path} ({len(overrides)} tier overrides)")


if __name__ == "__main__":
    main()