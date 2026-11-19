#!/usr/bin/env python3
"""Report R18 narrow (viewport width <= 390) coverage from manifest (v1.21)."""
import json
import sys
from pathlib import Path

NARROW_MAX_W = 390


def main() -> None:
    manifest_path = Path(sys.argv[1])
    out_path = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("/tmp/r18-narrow-coverage.json")
    manifest = json.loads(manifest_path.read_text())
    spots = manifest["spots"]
    narrow = [s for s in spots if s.get("viewport", [1280])[0] <= NARROW_MAX_W]
    report = {
        "total_spots": len(spots),
        "narrow_spots": len(narrow),
        "narrow_ratio": round(len(narrow) / len(spots), 4) if spots else 0,
        "narrow_slugs": [s["slug"] for s in narrow],
        "notes": "Inventory only; not a product sign-off",
    }
    out_path.write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps(report, ensure_ascii=False))
    if len(narrow) < 1:
        raise SystemExit("no narrow spots in manifest")


if __name__ == "__main__":
    main()