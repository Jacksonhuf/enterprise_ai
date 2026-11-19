#!/usr/bin/env python3
"""Report R18 per-spot pixel tolerances above default (v1.22 inventory)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CONFIG = ROOT / "frontend/r18-baselines/max-diff-ratio.json"


def main() -> None:
    out = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("/tmp/r18-pixel-tolerance-audit.json")
    cfg = json.loads(CONFIG.read_text())
    default = float(cfg.get("default", 0.02))
    elevated = []
    for slug, ratio in sorted(cfg.items()):
        if slug == "default":
            continue
        ratio = float(ratio)
        if ratio > default:
            elevated.append(
                {
                    "slug": slug,
                    "max_diff_ratio": ratio,
                    "above_default_by": round(ratio - default, 4),
                }
            )
    payload = {
        "default_max_diff_ratio": default,
        "elevated_spot_count": len(elevated),
        "elevated_spots": elevated,
        "notes": "Inventory only; tighten only with green CI pixel gate",
    }
    out.write_text(json.dumps(payload, indent=2) + "\n")
    print(json.dumps(payload))
    print(f"Wrote {out}")


if __name__ == "__main__":
    main()