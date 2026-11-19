#!/usr/bin/env python3
"""Roll up R18 tolerance tightening manifests into segmented artifact (v1.38)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RATIOS = ROOT / "frontend/r18-baselines/max-diff-ratio.json"
FIXTURES = ROOT / "docs/development/fixtures"


def main() -> None:
    out = (
        Path(sys.argv[1])
        if len(sys.argv) > 1
        else Path("/tmp/r18-tolerance-tightening-rollup-v1.42.json")
    )
    steps = []
    for minor in range(23, 43):
        tag = f"v1.{minor}"
        path = FIXTURES / f"r18-tolerance-tightening-{tag}.json"
        if not path.exists():
            continue
        plan = json.loads(path.read_text())
        steps.append(
            {
                "manifest": tag,
                "disclaimer": plan.get("disclaimer", ""),
                "rationale": plan.get("rationale", ""),
                "tightened": plan.get("tightened", []),
            }
        )
    cfg = json.loads(RATIOS.read_text())
    early = [s for s in steps if s["manifest"] in {f"v1.{m}" for m in range(23, 29)}]
    late = [s for s in steps if s not in early]
    payload = {
        "rollup_status": "TIGHTENING_CLOSED_THROUGH_V142",
        "closed_through_manifest": "v1.42",
        "engineering_inventory_v138": "r18-tolerance-tightening-candidates-v1.38.json",
        "manifest_step_count": len(steps),
        "manifest_steps": steps,
        "archive_segments": [
            {
                "label": "v1.23-v1.28",
                "manifest_step_count": len(early),
                "manifests": [s["manifest"] for s in early],
            },
            {
                "label": "v1.29-v1.37",
                "manifest_step_count": len(late),
                "manifests": [s["manifest"] for s in late],
            },
            {
                "label": "v1.38-inventory",
                "manifest_step_count": 0,
                "manifests": [],
                "notes": "Tightening candidate inventory only",
            },
        ],
        "current_max_diff_ratio": cfg,
        "notes": "Segmented archive; hold manifests (e.g. v1.34 services) count in rollup",
    }
    out.write_text(json.dumps(payload, indent=2) + "\n")
    print(f"OK: tolerance rollup ({len(steps)} steps)")
    print(f"Wrote {out}")


if __name__ == "__main__":
    main()