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
    batch_status = doc.get("batch_status")
    if batch_status not in (
        "R18_NARROW_OPEN_BATCH_WAIT_V163",
        "R18_NARROW_OPEN_BATCH_WAIT_V164",
        "R18_NARROW_OPEN_BATCH_WAIT_V165",
    ):
        raise SystemExit("batch_status must be V163, V164, or V165")
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
    if batch_status in (
        "R18_NARROW_OPEN_BATCH_WAIT_V164",
        "R18_NARROW_OPEN_BATCH_WAIT_V165",
    ):
        prior_name = doc.get("prior_batch_fixture", "")
        prior = json.loads((FX / prior_name).read_text())
        prior_slugs = set(prior.get("batch_slugs", []))
        if batch_status == "R18_NARROW_OPEN_BATCH_WAIT_V164":
            if "v1.63" not in prior_name:
                raise SystemExit("V164 requires prior_batch_fixture v1.63")
            if prior_slugs & set(batch):
                raise SystemExit("batch 2 must not overlap batch 1 slugs")
            prior_batch = json.loads((FX / prior_name).read_text())
            batch1 = set(prior_batch.get("batch_slugs", []))
            remain = len(open_slugs) - len(batch1) - len(batch)
        else:
            if "v1.64" not in prior_name:
                raise SystemExit("V165 requires prior_batch_fixture v1.64")
            b164 = json.loads((FX / prior_name).read_text())
            b163_name = b164.get("prior_batch_fixture", "")
            b163 = json.loads((FX / b163_name).read_text())
            used = set(b163.get("batch_slugs", [])) | set(b164.get("batch_slugs", []))
            if used & set(batch):
                raise SystemExit("batch 3 must not overlap prior batches")
            remain = len(open_slugs) - len(used) - len(batch)
    else:
        remain = len(open_slugs) - len(batch)
    if remain != int(doc.get("remaining_open_after_batch", -1)):
        raise SystemExit("remaining_open_after_batch mismatch")
    print(f"OK: narrow open batch wait ({len(batch)} slugs, {remain} remain)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()