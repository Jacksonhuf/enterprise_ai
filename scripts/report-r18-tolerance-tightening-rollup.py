#!/usr/bin/env python3
"""Roll up R18 tolerance tightening manifests v1.23–v1.31 into one artifact (v1.32)."""
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
        else Path("/tmp/r18-tolerance-tightening-rollup-v1.32.json")
    )
    steps = []
    for minor in range(23, 32):
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
    payload = {
        "rollup_status": "TIGHTENING_CLOSED_THROUGH_V131",
        "closed_through_manifest": "v1.31",
        "manifest_step_count": len(steps),
        "manifest_steps": steps,
        "current_max_diff_ratio": cfg,
        "notes": "Inventory for release archive; further tightening starts new manifest v1.32+",
    }
    out.write_text(json.dumps(payload, indent=2) + "\n")
    print(f"OK: tolerance rollup ({len(steps)} steps)")
    print(f"Wrote {out}")


if __name__ == "__main__":
    main()