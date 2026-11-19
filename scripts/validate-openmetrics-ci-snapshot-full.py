#!/usr/bin/env python3
"""Verify CI OpenMetrics snapshot: 6 metrics + tier/runbook/mixed summaries (v1.22)."""
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def metric_names(text: str) -> set[str]:
    return set(re.findall(r"^(cognara_fg_\w+)\s", text, re.MULTILINE))


def main() -> None:
    if len(sys.argv) != 5:
        raise SystemExit(
            "usage: validate-openmetrics-ci-snapshot-full.py "
            "<tier-summary.json> <runbook-summary.json> <mixed-read-summary.json> "
            "<openmetrics-ci-snapshot.txt>"
        )
    tier_path, runbook_path, mixed_path, om_path = map(Path, sys.argv[1:5])
    for p in (tier_path, runbook_path, mixed_path, om_path):
        if not p.is_file():
            raise SystemExit(f"missing file: {p}")

    draft = (ROOT / "docs/development/fixtures/openmetrics-exporter-draft.txt").read_text()
    want = metric_names(draft)
    text = om_path.read_text()
    if "OPENMETRICS_TIER_LINKED" not in text and "OPENMETRICS_EXPORTER_DRAFT_ONLY" not in text:
        raise SystemExit("snapshot missing OPENMETRICS_TIER_LINKED banner")
    got = metric_names(text)
    if got != want:
        raise SystemExit(f"metric set mismatch want={sorted(want)} got={sorted(got)}")
    if len(got) != 6:
        raise SystemExit(f"expected 6 metrics, got {len(got)}")

    scripts = ROOT / "scripts"
    for script, args in (
        ("validate-openmetrics-tier-linkage.py", [tier_path, om_path]),
        ("validate-openmetrics-runbook-linkage.py", [runbook_path, om_path]),
        ("validate-openmetrics-mixed-read-linkage.py", [mixed_path, om_path]),
    ):
        subprocess.check_call([sys.executable, str(scripts / script), *[str(a) for a in args]])

    print("OK: openmetrics ci snapshot full linkage (6 metrics)")
    print(f"Validated {om_path}")


if __name__ == "__main__":
    main()