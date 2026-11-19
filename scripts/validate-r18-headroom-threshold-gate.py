#!/usr/bin/env python3
"""Fail CI rehearsal when measured headroom drops below threshold (v1.36)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_POLICY = ROOT / "docs/development/fixtures/r18-headroom-threshold-gate-v1.36.json"


def main() -> None:
    audit_path = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("/tmp/r18-elevated-headroom-audit.json")
    policy_path = Path(sys.argv[2]) if len(sys.argv) > 2 else DEFAULT_POLICY
    audit = json.loads(audit_path.read_text())
    policy = json.loads(policy_path.read_text())
    min_h = float(policy.get("min_headroom_ratio", 0.002))
    violations = []
    for row in audit.get("spots", []):
        headroom = row.get("headroom")
        if headroom is None:
            continue
        if float(headroom) < min_h:
            violations.append(
                {
                    "slug": row["slug"],
                    "headroom": headroom,
                    "min_headroom_ratio": min_h,
                    "measured_ratio": row.get("measured_ratio"),
                }
            )
    enforce = bool(policy.get("enforce_in_ci", True))
    if violations:
        print(json.dumps({"violations": violations}, indent=2))
        if enforce:
            raise SystemExit(f"headroom threshold gate: {len(violations)} spot(s) below {min_h}")
        print(
            f"WARN: headroom threshold rehearsal ({len(violations)} spot(s) below {min_h}; "
            "enforce_in_ci=false)"
        )
    measured_count = sum(1 for r in audit.get("spots", []) if r.get("headroom") is not None)
    print(f"OK: headroom threshold gate (min={min_h}, measured_spots={measured_count})")
    print(f"Validated {audit_path}")


if __name__ == "__main__":
    main()