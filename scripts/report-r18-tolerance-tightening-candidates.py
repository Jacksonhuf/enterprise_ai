#!/usr/bin/env python3
"""Report R18 elevated/narrow spots as next tightening candidates (v1.38)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"
MANIFEST = ROOT / "frontend/r18-baselines/manifest.json"
POLICY = ROOT / "docs/development/fixtures/r18-tolerance-tightening-candidates-v1.38.json"


def main() -> None:
    audit_path = (
        Path(sys.argv[1])
        if len(sys.argv) > 1
        else Path("/tmp/r18-elevated-headroom-audit.json")
    )
    out = (
        Path(sys.argv[2])
        if len(sys.argv) > 2
        else Path("/tmp/r18-tolerance-tightening-candidates.json")
    )
    cfg = json.loads(RATIOS.read_text())
    default = float(cfg.get("default", 0.02))
    slugs = {s["slug"] for s in json.loads(MANIFEST.read_text())["spots"]}
    policy = json.loads(POLICY.read_text())
    exclude = set(policy.get("exclude_from_automatic_tighten", []))
    measured = {}
    if audit_path.is_file():
        for row in json.loads(audit_path.read_text()).get("spots", []):
            measured[row["slug"]] = row
    rows = []
    for slug in sorted(slugs):
        max_r = float(cfg.get(slug, default))
        is_narrow = slug.endswith("-narrow")
        elevated = max_r > default
        if not (is_narrow or elevated):
            continue
        m = measured.get(slug, {})
        rows.append(
            {
                "slug": slug,
                "max_diff_ratio": max_r,
                "narrow": is_narrow,
                "elevated_above_default": elevated,
                "measured_ratio": m.get("measured_ratio"),
                "headroom": m.get("headroom"),
                "excluded": slug in exclude,
            }
        )
    with_headroom = [r for r in rows if r.get("headroom") is not None]
    with_headroom.sort(key=lambda r: float(r["headroom"]))
    ranked = with_headroom + [r for r in rows if r.get("headroom") is None]
    payload = {
        "inventory_status": "TIGHTENING_CANDIDATE_INVENTORY_V138",
        "candidate_count": len(rows),
        "measured_candidate_count": len(with_headroom),
        "excluded_slugs": sorted(exclude),
        "pinned_priority_slugs": policy.get("pinned_priority_slugs", []),
        "ranked_candidates": ranked,
        "notes": "No tightening applied in v1.38; use after green pixel gate",
    }
    out.write_text(json.dumps(payload, indent=2) + "\n")
    print(f"OK: tightening candidates ({len(rows)} elevated/narrow, {len(with_headroom)} measured)")
    print(f"Wrote {out}")


if __name__ == "__main__":
    main()