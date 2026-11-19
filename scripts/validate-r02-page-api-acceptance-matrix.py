#!/usr/bin/env python3
"""Validate R02 page/API acceptance matrix fixture against router, views, and R18 manifest (v1.43)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ROUTER = ROOT / "frontend/src/cognara/router.ts"
MANIFEST = ROOT / "frontend/r18-baselines/manifest.json"


def route_patterns(router_text: str) -> list[str]:
    paths: list[str] = []
    for line in router_text.splitlines():
        line = line.strip()
        if line.startswith('path: "'):
            paths.append(line.split('"')[1])
    return paths


def main() -> None:
    path = Path(sys.argv[1])
    matrix = json.loads(path.read_text())
    status = matrix.get("matrix_status")
    if status not in (
        "R02_PAGE_API_ACCEPTANCE_MATRIX_V143",
        "R02_PAGE_API_ACCEPTANCE_MATRIX_V145",
        "R02_PAGE_API_ACCEPTANCE_MATRIX_V149",
    ):
        raise SystemExit("matrix_status must be V143, V145, or V149")
    entries = matrix.get("entries", [])
    expected = int(matrix.get("design_entry_count", 17))
    if len(entries) != expected:
        raise SystemExit(f"entries count {len(entries)} != design_entry_count {expected}")
    ids = [e["design_id"] for e in entries]
    if len(ids) != len(set(ids)):
        raise SystemExit("duplicate design_id in entries")
    manifest_slugs = {s["slug"] for s in json.loads(MANIFEST.read_text())["spots"]}
    expected_spots = int(matrix.get("r18_spot_count_expected", 38))
    if len(manifest_slugs) != expected_spots:
        raise SystemExit(f"manifest spot count {len(manifest_slugs)} != {expected_spots}")
    router_text = ROUTER.read_text()
    patterns = route_patterns(router_text)
    for entry in entries:
        for route in entry.get("routes", []):
            base = route.split("?")[0].split(":")[0].rstrip("/") or "/"
            if base == "/":
                continue
            if not any(p.startswith(base) or base.startswith(p.rstrip("/")) for p in patterns):
                raise SystemExit(f"{entry['design_id']}: route not in router: {route}")
        for slug in entry.get("r18_slugs", []):
            if slug not in manifest_slugs:
                raise SystemExit(f"{entry['design_id']}: unknown r18 slug {slug}")
        for view in entry.get("views", []):
            vp = ROOT / view
            if not vp.is_file():
                raise SystemExit(f"{entry['design_id']}: missing view file {view}")
    def ingest_extra(block: dict | None, label: str) -> None:
        if not block:
            return
        for slug in block.get("r18_slugs", []):
            if slug not in manifest_slugs:
                raise SystemExit(f"{label}: unknown r18 slug {slug}")
        for view in block.get("views", []):
            if view and not (ROOT / view).is_file():
                raise SystemExit(f"{label}: missing view {view}")

    ingest_extra(matrix.get("settings_sections_entry"), "settings_sections")
    for i, block in enumerate(matrix.get("supplementary_r18_mappings", [])):
        ingest_extra(block, f"supplementary[{i}]")
    all_matrix_slugs: set[str] = set()
    for entry in entries:
        all_matrix_slugs.update(entry.get("r18_slugs", []))
    extra = matrix.get("settings_sections_entry")
    if extra:
        all_matrix_slugs.update(extra.get("r18_slugs", []))
    for block in matrix.get("supplementary_r18_mappings", []):
        all_matrix_slugs.update(block.get("r18_slugs", []))
    uncovered = manifest_slugs - all_matrix_slugs
    if uncovered:
        raise SystemExit(f"R18 slugs not mapped to any design entry: {sorted(uncovered)}")
    print(f"OK: R02 matrix ({len(entries)} design entries, {len(all_matrix_slugs)} R18 slugs mapped)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()