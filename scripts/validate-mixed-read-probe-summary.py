#!/usr/bin/env python3
"""Validate mixed read probe summary JSON shape (v1.21)."""
import json
import sys
from pathlib import Path

REQUIRED = frozenset(
    {
        "disclaimer",
        "generated_at_utc",
        "routes",
        "workers",
        "elapsed",
        "source_test",
        "notes",
    }
)


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if set(data.keys()) != REQUIRED:
        raise SystemExit(f"keys must be {sorted(REQUIRED)}")
    if data["disclaimer"] != "MIXED_READ_PROBE_SUMMARY_NOT_SLO":
        raise SystemExit("invalid disclaimer")
    if data["routes"] != 6 or data["workers"] != 36:
        raise SystemExit("routes/workers must be 6/36")
    print(f"OK: mixed read summary shape")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()