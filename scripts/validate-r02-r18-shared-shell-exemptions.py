#!/usr/bin/env python3
"""Validate R18 shared-shell exemptions against R02 matrix entries (v1.44)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FIXTURES = ROOT / "docs/development/fixtures"


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("exemption_status") != "R18_SHARED_SHELL_EXEMPTIONS_V144":
        raise SystemExit("exemption_status must be R18_SHARED_SHELL_EXEMPTIONS_V144")
    matrix_path = FIXTURES / doc.get("matrix_fixture", "")
    if not matrix_path.is_file():
        raise SystemExit(f"missing matrix fixture: {matrix_path.name}")
    matrix = json.loads(matrix_path.read_text())
    by_id = {e["design_id"]: e for e in matrix.get("entries", [])}
    manifest_slugs = {s["slug"] for s in json.loads((ROOT / "frontend/r18-baselines/manifest.json").read_text())["spots"]}
    for ex in doc.get("exemptions", []):
        did = ex.get("design_id")
        if did not in by_id:
            raise SystemExit(f"unknown design_id in exemptions: {did}")
        entry = by_id[did]
        shell = ex.get("shared_shell_slug")
        slugs = entry.get("r18_slugs", [])
        if shell not in slugs and shell not in manifest_slugs:
            raise SystemExit(f"{did}: shared_shell_slug {shell} not in entry slugs or manifest")
        gaps = entry.get("gaps", [])
        if "NO_R18_DEDICATED_SPOT" not in gaps and "NO_WIDE_R18_SPOT" not in gaps:
            raise SystemExit(f"{did}: matrix must list NO_R18_DEDICATED_SPOT or NO_WIDE_R18_SPOT")
    print(f"OK: shared-shell exemptions ({len(doc.get('exemptions', []))} entries)")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()