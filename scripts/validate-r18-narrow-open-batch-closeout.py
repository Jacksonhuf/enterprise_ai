#!/usr/bin/env python3
"""Closeout for R18 narrow open batch-wait sequence (v1.66)."""
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
    if doc.get("closeout_status") != "R18_NARROW_OPEN_BATCH_CLOSEOUT_V166":
        raise SystemExit("closeout_status must be R18_NARROW_OPEN_BATCH_CLOSEOUT_V166")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r18-narrow-only-defer-inventory.py"), str(FX / doc["inventory_fixture"])],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r18-publish-review-narrow-wait-decision.py"), str(FX / doc["publish_review_narrow_fixture"])],
        check=True,
        cwd=ROOT,
    )
    inv = json.loads((FX / doc["inventory_fixture"]).read_text())
    open_slugs = set(inv.get("open_narrow_without_defer_slugs", []))
    batched = set()
    for name in doc.get("batch_fixtures", []):
        subprocess.run(
            [PY, str(ROOT / "scripts/validate-r18-narrow-open-batch-wait.py"), str(FX / name)],
            check=True,
            cwd=ROOT,
        )
        batch = json.loads((FX / name).read_text())
        slugs = set(batch.get("batch_slugs", []))
        if batched & slugs:
            raise SystemExit(f"overlap in batches: {name}")
        batched |= slugs
    pr = "publish-review-narrow"
    if pr in batched:
        raise SystemExit("publish-review-narrow must not appear in batch fixtures")
    remain = open_slugs - batched
    expected = set(doc.get("expected_remaining_open_slugs", []))
    if remain != expected:
        raise SystemExit(f"remaining open mismatch: {sorted(remain)} vs {sorted(expected)}")
    if len(batched) != int(doc.get("expected_batched_open_count", 0)):
        raise SystemExit("expected_batched_open_count mismatch")
    closeout_pr = doc.get("publish_review_narrow_closeout_fixture", "")
    if "v1.66" not in closeout_pr:
        raise SystemExit("must reference publish-review-narrow closeout v1.66")
    print(f"OK: narrow batch closeout ({len(batched)}/{len(open_slugs)} batched)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()