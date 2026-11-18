#!/usr/bin/env python3
"""Extract runbook default smoke lines from go test output and write JSON summary."""
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

TEST = "TestR20Schema8RunbookDefaultSmoke"
log_path = Path(sys.argv[1]) if len(sys.argv) > 1 else None
out_path = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("/tmp/runbook-smoke-summary.json")
text = log_path.read_text() if log_path and log_path.is_file() else sys.stdin.read()

status = "unknown"
if re.search(rf"--- PASS: {re.escape(TEST)}", text):
    status = "pass"
elif re.search(rf"--- FAIL: {re.escape(TEST)}", text):
    status = "fail"
elif f"ok  	github.com" in text and TEST not in text:
    status = "included_in_package_ok"

write_elapsed = None
read_elapsed = None
for line in text.splitlines():
    if "write phase public+profile elapsed=" in line:
        m = re.search(r"elapsed=([\d.]+[a-zµ]+)", line)
        if m:
            write_elapsed = m.group(1)
    if "read phase workers=" in line and "elapsed=" in line:
        m = re.search(r"elapsed=([\d.]+[a-zµ]+)", line)
        if m:
            read_elapsed = m.group(1)

summary = {
    "test": TEST,
    "status": status,
    "generated_at_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "write_phase_elapsed": write_elapsed,
    "read_phase_elapsed": read_elapsed,
    "notes": "Attach this file or CI artifact to runbook checklist §4 when production apply smoke is recorded.",
}
out_path.write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n")
print(json.dumps(summary, ensure_ascii=False))
if status == "fail":
    raise SystemExit(1)