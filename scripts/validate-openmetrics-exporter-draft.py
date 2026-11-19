#!/usr/bin/env python3
"""Validate OpenMetrics exporter draft sample against recording rule expr (v1.16)."""
import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    yaml = None

ROOT = Path(__file__).resolve().parents[1]
REC = ROOT / "docs/development/fixtures/r21-recording-rules-draft.yaml"


def main() -> None:
    path = Path(sys.argv[1])
    text = path.read_text()
    if "OPENMETRICS_EXPORTER_DRAFT_ONLY" not in text and "OPENMETRICS_TIER_LINKED" not in text:
        raise SystemExit("missing OPENMETRICS_EXPORTER_DRAFT_ONLY or OPENMETRICS_TIER_LINKED banner")
    if yaml is None:
        raise SystemExit("PyYAML required")
    doc = yaml.safe_load(REC.read_text())
    want = {r["expr"] for r in doc["groups"][0]["rules"]}
    got = set(re.findall(r"^(cognara_fg_\w+)\s", text, re.MULTILINE))
    if got != want:
        raise SystemExit(f"metric mismatch want={sorted(want)} got={sorted(got)}")
    print(f"OK: openmetrics exporter draft ({len(got)} metrics)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()