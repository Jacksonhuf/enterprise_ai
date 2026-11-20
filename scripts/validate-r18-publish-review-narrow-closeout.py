#!/usr/bin/env python3
"""Closeout registration for publish-review-narrow wait decision (v1.66)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("closeout_status") != "R18_PUBLISH_REVIEW_NARROW_CLOSEOUT_V166":
        raise SystemExit("closeout_status must be R18_PUBLISH_REVIEW_NARROW_CLOSEOUT_V166")
    prior = doc.get("prior_wait_fixture", "")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r18-publish-review-narrow-wait-decision.py"), str(FX / prior)],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r18-narrow-open-batch-closeout.py"), str(FX / doc["narrow_batch_closeout_fixture"])],
        check=True,
        cwd=ROOT,
    )
    wait = json.loads((FX / prior).read_text())
    if doc.get("resolution") != wait.get("resolution"):
        raise SystemExit("closeout must keep same resolution as wait fixture")
    if not doc.get("resolution_unchanged"):
        raise SystemExit("resolution_unchanged must be true")
    if doc.get("slug") != wait.get("slug"):
        raise SystemExit("slug mismatch")
    if not doc.get("only_remaining_open_narrow"):
        raise SystemExit("only_remaining_open_narrow must be true")
    print("OK: publish-review-narrow closeout (wait decision unchanged)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()