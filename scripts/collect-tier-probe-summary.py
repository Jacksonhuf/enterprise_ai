#!/usr/bin/env python3
"""Extract R09 tier probe elapsed lines from harness log (v1.16)."""
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

PATTERNS = (
    ("message_load", re.compile(r"message load tier workers=(\d+) elapsed=([\d.]+[a-zµ]+)")),
    ("governance_usage", re.compile(r"governance usage tier workers=(\d+) elapsed=([\d.]+[a-zµ]+)")),
)


def main() -> None:
    log_path = Path(sys.argv[1])
    out_path = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("/tmp/r09-tier-probe-summary.json")
    text = log_path.read_text()
    tiers = []
    for kind, pat in PATTERNS:
        for m in pat.finditer(text):
            tiers.append({"probe": kind, "workers": int(m.group(1)), "elapsed": m.group(2)})
    summary = {
        "disclaimer": "TIER_PROBE_SUMMARY_NOT_SLO",
        "generated_at_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "tiers": tiers,
        "notes": "Attach CI artifact r09-tier-probe-summary to engineering pilot ledger; not a signed SLO.",
    }
    out_path.write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps(summary, ensure_ascii=False))
    if not tiers:
        raise SystemExit("no tier lines found in log")


if __name__ == "__main__":
    main()