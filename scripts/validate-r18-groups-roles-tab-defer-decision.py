#!/usr/bin/env python3
"""Written decision for CAND-09/11 groups/roles R18 slugs (v1.60)."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
MANIFEST = ROOT / "frontend/r18-baselines/manifest.json"


def main() -> None:
    path = Path(sys.argv[1])
    doc = json.loads(path.read_text())
    if doc.get("decision_status") != "R18_GROUPS_ROLES_TAB_DEFER_V160":
        raise SystemExit("decision_status must be R18_GROUPS_ROLES_TAB_DEFER_V160")
    if doc.get("resolution") != "DEDICATED_SLUGS_ACCEPTED":
        raise SystemExit("resolution must be DEDICATED_SLUGS_ACCEPTED")
    matrix = json.loads((FX / doc["matrix_fixture"]).read_text())
    by_id = {e["design_id"]: e for e in matrix.get("entries", [])}
    slug_by_design = {
        "CAND-09": "governance-access-groups",
        "CAND-11": "governance-access-roles",
    }
    for cid in doc.get("design_ids", []):
        entry = by_id.get(cid)
        if not entry:
            raise SystemExit(f"missing matrix entry {cid}")
        slugs = set(entry.get("r18_slugs", []))
        need = slug_by_design.get(cid)
        if need and need not in slugs:
            raise SystemExit(f"{cid} must list slug {need}")
    manifest = json.loads(MANIFEST.read_text())
    manifest_slugs = {s["slug"] for s in manifest.get("spots", [])}
    for need in doc.get("manifest_slugs", []):
        if need not in manifest_slugs:
            raise SystemExit(f"manifest missing slug {need}")
    gap_doc = ROOT / "docs/development" / doc["r18_gap_doc"]
    if not gap_doc.is_file():
        raise SystemExit("missing r18 gap doc")
    print(f"OK: groups/roles tab defer ({', '.join(doc['design_ids'])})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()