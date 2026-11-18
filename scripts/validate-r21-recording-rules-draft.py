#!/usr/bin/env python3
"""Validate R21 recording rules draft YAML against metrics wiring (v1.15)."""
import json
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    yaml = None

ROOT = Path(__file__).resolve().parents[1]
WIRING = ROOT / "docs/development/fixtures/r21-metrics-wiring-draft.json"


def main() -> None:
    path = Path(sys.argv[1])
    text = path.read_text()
    if "RECORDING_RULES_DRAFT_ONLY" not in text:
        raise SystemExit("missing RECORDING_RULES_DRAFT_ONLY banner")
    if yaml is None:
        raise SystemExit("PyYAML required")
    doc = yaml.safe_load(text)
    rules = doc["groups"][0]["rules"]
    records = {r["record"] for r in rules}
    wiring = json.loads(WIRING.read_text())
    want = {b["recording_rule_hint"] for b in wiring["bindings"]}
    if records != want:
        raise SystemExit(f"record mismatch want={sorted(want)} got={sorted(records)}")
    print(f"OK: recording rules draft ({len(rules)} rules)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()