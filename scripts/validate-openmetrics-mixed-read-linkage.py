#!/usr/bin/env python3
"""Verify mixed-read metric in OpenMetrics matches probe summary (v1.21)."""
import json
import re
import sys
from pathlib import Path

METRIC = "cognara_fg_mixed_read_smoke_seconds"


def to_seconds(elapsed: str) -> float:
    if elapsed.endswith("ms"):
        return float(elapsed[:-2]) / 1000.0
    if elapsed.endswith("µs"):
        return float(elapsed[:-2]) / 1_000_000.0
    if elapsed.endswith("s"):
        return float(elapsed[:-1])
    raise ValueError(elapsed)


def main() -> None:
    summary_path = Path(sys.argv[1])
    om_path = Path(sys.argv[2])
    data = json.loads(summary_path.read_text())
    exp = to_seconds(data["elapsed"])
    text = om_path.read_text()
    m = re.search(rf"^{METRIC}\s+([\d.eE+-]+)\s*$", text, re.MULTILINE)
    if not m:
        raise SystemExit(f"missing {METRIC}")
    got = float(m.group(1))
    if abs(got - exp) > 0.001:
        raise SystemExit(f"{METRIC}: want {exp} got {got}")
    print("OK: openmetrics mixed-read linkage")
    print(f"Validated {om_path} against {summary_path}")


if __name__ == "__main__":
    main()