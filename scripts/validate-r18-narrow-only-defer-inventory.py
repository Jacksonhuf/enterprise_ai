#!/usr/bin/env python3
"""Inventory manifest narrow slugs vs headroom defer rollup (v1.61)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    inv = json.loads(path.read_text())
    if inv.get("inventory_status") != "R18_NARROW_ONLY_DEFER_INVENTORY_V161":
        raise SystemExit("inventory_status must be R18_NARROW_ONLY_DEFER_INVENTORY_V161")
    rollup_name = inv.get("defer_rollup_fixture", "")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r18-headroom-defer-rollup.py"), str(FX / rollup_name)],
        check=True,
        cwd=ROOT,
    )
    rollup = json.loads((FX / rollup_name).read_text())
    defer_slugs = {r["slug"] for r in rollup.get("defer_records", [])}
    manifest = json.loads((ROOT / inv["manifest_path"]).read_text())
    narrow = sorted(s["slug"] for s in manifest.get("spots", []) if s["slug"].endswith("-narrow"))
    if len(narrow) != int(inv.get("expected_manifest_narrow_count", 0)):
        raise SystemExit("manifest narrow count mismatch")
    want_def = set(inv.get("deferred_narrow_slugs", []))
    for slug in want_def:
        if slug not in defer_slugs:
            raise SystemExit(f"rollup missing deferred narrow {slug}")
    for slug in inv.get("deferred_non_narrow_slugs", []):
        if slug not in defer_slugs:
            raise SystemExit(f"rollup missing deferred slug {slug}")
    open_slugs = sorted(set(narrow) - want_def)
    fixture_open = inv.get("open_narrow_without_defer_slugs", [])
    if open_slugs != sorted(fixture_open):
        raise SystemExit(f"open narrow list mismatch: got {open_slugs}")
    if len(open_slugs) != int(inv.get("expected_open_narrow_without_defer_count", 0)):
        raise SystemExit("open narrow count mismatch")
    print(f"OK: narrow-only defer inventory ({len(want_def)} deferred, {len(open_slugs)} open)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()