#!/usr/bin/env python3
"""Overlay runbook smoke read_phase onto OpenMetrics text (v1.20)."""
import json
import re
import sys
from pathlib import Path

METRIC = "cognara_fg_runbook_smoke_read_seconds"


def to_seconds(elapsed: str) -> float:
    if not elapsed:
        raise SystemExit("runbook summary missing read_phase_elapsed")
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
    status = data.get("status")
    if status not in ("pass", "included_in_package_ok"):
        raise SystemExit(f"runbook summary status invalid: {status}")
    seconds = to_seconds(data["read_phase_elapsed"])
    text = in_path.read_text()
    lines = []
    for line in text.splitlines():
        m = re.match(r"^(cognara_fg_runbook_smoke_read_seconds)\s+([\d.eE+-]+)\s*$", line)
        if m:
            lines.append(f"{METRIC} {seconds:.6g}")
        else:
            lines.append(line)
    if not any(line.startswith(METRIC) for line in lines):
        raise SystemExit("input missing runbook smoke metric line")
    header = "# OPENMETRICS_RUNBOOK_LINKED — runbook read_phase from CI summary\n"
    if text.startswith("# OPENMETRICS"):
        body = "\n".join(lines)
    else:
        body = header + "\n".join(lines)
    out_path.write_text(body if body.endswith("\n") else body + "\n")
    print(f"Wrote {out_path} (runbook read {seconds:.6g}s)")


if __name__ == "__main__":
    main()