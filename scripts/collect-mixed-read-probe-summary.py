#!/usr/bin/env python3
"""Extract mixed read smoke wall time from go test -v log (v1.21)."""
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

PAT = re.compile(
    r"mixed read-only smoke routes=(\d+) workers=(\d+) elapsed=([\d.]+[a-zµ]+)"
)


def main() -> None:
    log_path = Path(sys.argv[1])
    out_path = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("/tmp/mixed-read-probe-summary.json")
    text = log_path.read_text()
    m = PAT.search(text)
    if not m:
        raise SystemExit("mixed read smoke line not found in log")
    summary = {
        "disclaimer": "MIXED_READ_PROBE_SUMMARY_NOT_SLO",
        "generated_at_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "routes": int(m.group(1)),
        "workers": int(m.group(2)),
        "elapsed": m.group(3),
        "source_test": "TestR20Schema8MixedReadOnlySmokeOrchestration",
        "notes": "Feeds openmetrics-ci-snapshot mixed_read metric",
    }
    out_path.write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps(summary, ensure_ascii=False))
    print(f"Wrote {out_path}")


if __name__ == "__main__":
    main()