#!/usr/bin/env python3
"""Validate R21 metrics wiring draft JSON against dry-run rule metrics (v1.14)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DRY = ROOT / "docs/development/fixtures/r21-alerting-dry-run.rules.json"
REQ = frozenset({"metric", "producer", "scrape_job_placeholder", "recording_rule_hint"})


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if data.get("disclaimer") != "METRICS_WIRING_DRAFT_ONLY":
        raise SystemExit("disclaimer must be METRICS_WIRING_DRAFT_ONLY")
    dry = json.loads(DRY.read_text())
    want = {r["metric"] for r in dry["rules"]}
    bindings = data.get("bindings")
    if not isinstance(bindings, list):
        raise SystemExit("bindings must be a list")
    got = set()
    for b in bindings:
        if set(b.keys()) != REQ:
            raise SystemExit(f"binding keys must be {sorted(REQ)}")
        got.add(b["metric"])
    if got != want:
        missing = want - got
        extra = got - want
        raise SystemExit(f"metric mismatch missing={sorted(missing)} extra={sorted(extra)}")
    print(f"OK: metrics wiring draft ({len(bindings)} bindings)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()