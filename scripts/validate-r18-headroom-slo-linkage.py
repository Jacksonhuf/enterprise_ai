#!/usr/bin/env python3
"""Validate R18 headroom ↔ SLO linkage fixture (v1.37)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs/development"
FIXTURES = DOCS / "fixtures"


def main() -> None:
    path = Path(sys.argv[1])
    link = json.loads(path.read_text())
    if link.get("linkage_status") != "HEADROOM_SLO_REHEARSAL_V137":
        raise SystemExit("linkage_status must be HEADROOM_SLO_REHEARSAL_V137")
    gate_name = link.get("headroom_gate_fixture", "")
    if not (FIXTURES / gate_name).is_file():
        raise SystemExit(f"missing headroom gate fixture: {gate_name}")
    for doc in link.get("slo_documents", []):
        if not (DOCS / doc).is_file():
            raise SystemExit(f"missing slo linkage doc: {doc}")
    print("OK: R18 headroom SLO linkage")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()