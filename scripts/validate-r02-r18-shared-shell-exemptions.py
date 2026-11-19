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
    status = doc.get("exemption_status")
    if status not in (
        "R18_SHARED_SHELL_EXEMPTIONS_V144",
        "R18_SHARED_SHELL_EXEMPTIONS_RESOLVED_V145",
    ):
        raise SystemExit("invalid exemption_status")
    matrix_path = FIXTURES / doc.get("matrix_fixture", "")
    if not matrix_path.is_file():
        raise SystemExit(f"missing matrix fixture: {matrix_path.name}")
    matrix = json.loads(matrix_path.read_text())
    by_id = {e["design_id"]: e for e in matrix.get("entries", [])}
    manifest_slugs = {s["slug"] for s in json.loads((ROOT / "frontend/r18-baselines/manifest.json").read_text())["spots"]}
    if status == "R18_SHARED_SHELL_EXEMPTIONS_RESOLVED_V145":
        if doc.get("exemptions"):
            raise SystemExit("resolved v1.45 must have empty exemptions")
        resolved = doc.get("resolved_in_v145", [])
        if len(resolved) != 3:
            raise SystemExit("resolved_in_v145 must list 3 slugs")
        matrix_path = FIXTURES / doc.get("matrix_fixture", "")
        slugs = {
            s["slug"]
            for s in json.loads(
                (ROOT / "frontend/r18-baselines/manifest.json").read_text()
            )["spots"]
        }
        for item in resolved:
            slug = item.split(":", 1)[-1]
            if slug not in slugs:
                raise SystemExit(f"resolved slug missing from manifest: {slug}")
        print(f"OK: shared-shell exemptions resolved ({len(resolved)} slugs)")
        print(f"Validated {path}")
        return
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