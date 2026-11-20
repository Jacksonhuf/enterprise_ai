#!/usr/bin/env python3
"""Batch wait decisions for open R18 narrow slugs (v1.63)."""
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
    if doc.get("batch_status") != "R18_NARROW_OPEN_BATCH_WAIT_V163":
        raise SystemExit("batch_status must be R18_NARROW_OPEN_BATCH_WAIT_V163")
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
    batch = doc.get("batch_slugs", [])
    if len(batch) != int(doc.get("expected_batch_count", 0)):
        raise SystemExit("batch count mismatch")
    for slug in batch:
        if slug not in open_slugs:
            raise SystemExit(f"{slug} not in open narrow inventory")
    pr_slug = "publish-review-narrow"
    if pr_slug in batch:
        raise SystemExit("publish-review-narrow must use dedicated wait fixture, not batch defer")
    remain = len(open_slugs) - len(batch)
    if remain != int(doc.get("remaining_open_after_batch", -1)):
        raise SystemExit("remaining_open_after_batch mismatch")
    print(f"OK: narrow open batch wait ({len(batch)} slugs, {remain} remain)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()