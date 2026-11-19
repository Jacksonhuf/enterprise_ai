#!/usr/bin/env python3
"""Enforce headroom threshold on a CI subset of spots (v1.40)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_POLICY = ROOT / "docs/development/fixtures/r18-headroom-enforce-subset-v1.40.json"


def main() -> None:
    audit_path = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("/tmp/r18-elevated-headroom-audit.json")
    policy_path = Path(sys.argv[2]) if len(sys.argv) > 2 else DEFAULT_POLICY
    audit = json.loads(audit_path.read_text())
    policy = json.loads(policy_path.read_text())
    if policy.get("subset_status") != "HEADROOM_ENFORCE_SUBSET_V140":
        raise SystemExit("subset_status must be HEADROOM_ENFORCE_SUBSET_V140")
    min_h = float(policy.get("min_headroom_ratio", 0.002))
    enforce_slugs = set(policy.get("enforce_slugs", []))
    failures = []
    warnings = []
    for row in audit.get("spots", []):
        slug = row.get("slug")
        headroom = row.get("headroom")
        if headroom is None:
            continue
        if float(headroom) < min_h:
            entry = {
                "slug": slug,
                "headroom": headroom,
                "min_headroom_ratio": min_h,
                "measured_ratio": row.get("measured_ratio"),
            }
            if slug in enforce_slugs:
                failures.append(entry)
            else:
                warnings.append(entry)
    if failures:
        print(json.dumps({"subset_enforce_failures": failures}, indent=2))
        raise SystemExit(f"headroom subset gate: {len(failures)} enforced slug(s) below {min_h}")
    if warnings:
        print(f"WARN: headroom subset gate ({len(warnings)} non-enforced slug(s) below {min_h})")
    print(f"OK: headroom subset gate (enforce={sorted(enforce_slugs)}, failures=0)")
    print(f"Validated {audit_path}")


if __name__ == "__main__":
    main()