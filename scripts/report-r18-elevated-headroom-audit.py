#!/usr/bin/env python3
"""Merge elevated tolerances with CI pixel gate measurements (v1.35)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"
MANIFEST = ROOT / "frontend/r18-baselines/manifest.json"


def main() -> None:
    measured_path = (
        Path(sys.argv[1])
        if len(sys.argv) > 1
        else Path("/tmp/r18-pixel-gate-measured.json")
    )
    out = (
        Path(sys.argv[2])
        if len(sys.argv) > 2
        else Path("/tmp/r18-elevated-headroom-audit.json")
    )
    cfg = json.loads(RATIOS.read_text())
    default = float(cfg.get("default", 0.02))
    manifest = json.loads(MANIFEST.read_text())
    slugs = [s["slug"] for s in manifest["spots"]]
    measured = {}
    if measured_path.is_file():
        data = json.loads(measured_path.read_text())
        for row in data.get("spots", []):
            measured[row["slug"]] = row
    rows = []
    for slug in slugs:
        max_r = float(cfg.get(slug, default))
        m = measured.get(slug, {})
        measured_r = m.get("measured_ratio")
        headroom = m.get("headroom")
        if measured_r is not None and headroom is None:
            headroom = round(max_r - measured_r, 6)
        rows.append(
            {
                "slug": slug,
                "max_diff_ratio": max_r,
                "uses_default": slug not in cfg,
                "elevated_above_default": max_r > default,
                "measured_ratio": measured_r,
                "headroom": headroom,
            }
        )
    elevated = [r for r in rows if r["elevated_above_default"]]
    tight = sorted(
        [r for r in rows if r.get("headroom") is not None],
        key=lambda r: r["headroom"],
    )[:5]
    payload = {
        "audit_status": "ELEVATED_HEADROOM_V135",
        "default_max_diff_ratio": default,
        "spot_count": len(rows),
        "elevated_count": len(elevated),
        "tightest_headroom_spots": tight,
        "spots": rows,
        "notes": "headroom from CI when COGNARA_R18_PIXEL_MEASURED is set",
    }
    out.write_text(json.dumps(payload, indent=2) + "\n")
    print(f"OK: elevated headroom audit ({len(rows)} spots, {len(elevated)} elevated)")
    print(f"Wrote {out}")


if __name__ == "__main__":
    main()