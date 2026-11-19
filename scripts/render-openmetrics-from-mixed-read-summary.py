#!/usr/bin/env python3
"""Overlay mixed-read elapsed onto OpenMetrics snapshot (v1.21)."""
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
    raise SystemExit(f"unsupported elapsed: {elapsed}")


def main() -> None:
    summary_path = Path(sys.argv[1])
    in_path = Path(sys.argv[2])
    out_path = Path(sys.argv[3]) if len(sys.argv) > 3 else in_path
    data = json.loads(summary_path.read_text())
    seconds = to_seconds(data["elapsed"])
    text = in_path.read_text()
    lines = []
    found = False
    for line in text.splitlines():
        if re.match(rf"^{METRIC}\s+", line):
            lines.append(f"{METRIC} {seconds:.6g}")
            found = True
        else:
            lines.append(line)
    if not found:
        raise SystemExit(f"input missing {METRIC}")
    banner = "# OPENMETRICS_MIXED_READ_LINKED — from mixed-read probe summary\n"
    if not text.startswith("# OPENMETRICS_MIXED_READ"):
        lines.insert(0, banner.rstrip())
    body = "\n".join(lines)
    out_path.write_text(body + ("\n" if not body.endswith("\n") else ""))
    print(f"Wrote {out_path} (mixed read {seconds:.6g}s)")


if __name__ == "__main__":
    main()