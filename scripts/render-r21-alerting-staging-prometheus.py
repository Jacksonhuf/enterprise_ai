#!/usr/bin/env python3
"""Render R21 dry-run JSON to Prometheus-style alert snippets for staging import (v1.08)."""
import json
import sys
from pathlib import Path


def main() -> None:
    src = Path(sys.argv[1])
    out = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("/tmp/r21-staging-alerts.prom.txt")
    data = json.loads(src.read_text())
    if data.get("disclaimer") != "ALERTING_DRY_RUN_ONLY":
        raise SystemExit("source must be ALERTING_DRY_RUN_ONLY")
    lines = [
        "# STAGING IMPORT ONLY — not applied without SRE signoff",
        f"# source: {src}",
        "",
    ]
    for rule in data["rules"]:
        lines.extend(
            [
                f"# {rule['source_test']} pilot={rule['pilot_observed_seconds']}s",
                f"ALERT {rule['name']}",
                f"  IF {rule['metric']} > {rule['threshold_seconds']}",
                f"  FOR 5m",
                f"  LABELS {{ severity=\"warning\", staging=\"true\" }}",
                f"  ANNOTATIONS {{",
                f"    summary = \"schema8 fg dry-run gate {rule['name']}\",",
                f"    description = \"wall clock exceeds {rule['threshold_seconds']}s\",",
                f"  }}",
                "",
            ]
        )
    out.write_text("\n".join(lines))
    print(f"Wrote {out} ({len(data['rules'])} rules)")


if __name__ == "__main__":
    main()