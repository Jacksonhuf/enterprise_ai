#!/usr/bin/env python3
"""Validate rendered staging Prometheus alert file (v1.09); import remains manual."""
import re
import sys
from pathlib import Path


def main() -> None:
    path = Path(sys.argv[1])
    text = path.read_text()
    if "STAGING IMPORT ONLY" not in text:
        raise SystemExit("missing STAGING IMPORT ONLY banner")
    alerts = re.findall(r"^ALERT (\w+)", text, re.MULTILINE)
    if len(alerts) != 6:
        raise SystemExit(f"expected 6 ALERT blocks, got {len(alerts)}")
    for name in alerts:
        block = text.split(f"ALERT {name}", 1)[1].split("ALERT ", 1)[0]
        if 'staging="true"' not in block:
            raise SystemExit(f"missing staging label on {name}")
    print(f"OK: staging alert render ({len(alerts)} rules)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()