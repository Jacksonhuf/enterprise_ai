#!/usr/bin/env python3
"""Ensure scripts-smoke R18 slugs match committed manifest and PNG baselines (v1.24)."""
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SMOKE = ROOT / "frontend/scripts-smoke.mjs"
BASE = ROOT / "frontend/r18-baselines"
MANIFEST = BASE / "manifest.json"


def main() -> None:
    text = SMOKE.read_text()
    smoke_slugs = re.findall(r"\{slug:'([^']+)'", text)
    if not smoke_slugs:
        raise SystemExit("no r18 slugs in scripts-smoke.mjs")
    data = json.loads(MANIFEST.read_text())
    manifest_slugs = [s["slug"] for s in data["spots"]]
    if smoke_slugs != manifest_slugs:
        raise SystemExit(
            f"slug order/count mismatch smoke={len(smoke_slugs)} manifest={len(manifest_slugs)}"
        )
    missing_png = [s for s in smoke_slugs if not (BASE / f"{s}.png").is_file()]
    if missing_png:
        raise SystemExit(f"missing baseline PNG: {missing_png[:5]}")
    print(f"OK: R18 manifest sync ({len(smoke_slugs)} spots)")
    print(f"Validated {SMOKE} vs {MANIFEST}")


if __name__ == "__main__":
    main()