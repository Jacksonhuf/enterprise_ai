#!/usr/bin/env python3
"""Engineering decision artifact for CAND-14 services wide R18 slug (v1.58)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
DOC = ROOT / "docs/development"
MANIFEST = ROOT / "frontend/r18-baselines/manifest.json"
PY = sys.executable


def main() -> None:
    path = Path(sys.argv[1])
    decision = json.loads(path.read_text())
    if decision.get("decision_status") != "R18_SERVICES_WIDE_SLUG_DECISION_V158":
        raise SystemExit("decision_status must be R18_SERVICES_WIDE_SLUG_DECISION_V158")
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r18-spot-manifest-sync.py")],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-r02-page-api-acceptance-matrix.py"), str(FX / decision["matrix_fixture"])],
        check=True,
        cwd=ROOT,
    )
    manifest = json.loads(MANIFEST.read_text())
    slugs = {s["slug"] for s in manifest.get("spots", [])}
    required = decision.get("required_manifest_slugs", [])
    if set(required) != set(required) & slugs:
        missing = set(required) - slugs
        raise SystemExit(f"manifest missing slugs: {sorted(missing)}")
    matrix = json.loads((FX / decision["matrix_fixture"]).read_text())
    entry = next((e for e in matrix.get("entries", []) if e.get("design_id") == decision.get("design_id")), None)
    if not entry:
        raise SystemExit(f"matrix missing {decision.get('design_id')}")
    matrix_slugs = set(entry.get("r18_slugs", []))
    if set(required) != matrix_slugs:
        raise SystemExit("matrix r18_slugs must match required_manifest_slugs")
    text = (DOC / decision["r18_gap_doc"]).read_text()
    if "services-narrow" not in text or "宽屏" not in text:
        raise SystemExit("r18 gap doc must discuss services narrow/wide")
    if not decision.get("product_signoff_still_required"):
        raise SystemExit("product_signoff_still_required must be true")
    print(f"OK: R18 services wide slug decision ({','.join(required)})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()