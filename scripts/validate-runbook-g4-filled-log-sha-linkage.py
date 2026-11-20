#!/usr/bin/env python3
"""Link G4 exporter appendix SHA with filled rehearsal log digest (v1.60)."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FX = ROOT / "docs/development/fixtures"
DOC = ROOT / "docs/development"
PY = sys.executable


def dig(log: dict, dotted: str) -> str:
    cur: object = log
    for part in dotted.split("."):
        if not isinstance(cur, dict) or part not in cur:
            raise SystemExit(f"missing log path {dotted}")
        cur = cur[part]
    if not isinstance(cur, str) or len(cur) != 64:
        raise SystemExit(f"SHA at {dotted} must be 64 hex chars")
    return cur


def main() -> None:
    path = Path(sys.argv[1])
    link = json.loads(path.read_text())
    if link.get("linkage_status") != "RUNBOOK_G4_FILLED_LOG_SHA_V160":
        raise SystemExit("linkage_status must be RUNBOOK_G4_FILLED_LOG_SHA_V160")
    g4_doc = link.get("g4_link_doc", "")
    if not (DOC / g4_doc).is_file():
        raise SystemExit(f"missing G4 link doc {g4_doc}")
    log = json.loads((FX / link["filled_log_fixture"]).read_text())
    want = dig(log, link["sha_source_path"])
    appendix = json.loads((FX / link["g4_appendix_fixture"]).read_text())
    subprocess.run(
        [PY, str(ROOT / "scripts/validate-production-exporter-review.py"), str(FX / link["g4_appendix_fixture"])],
        check=True,
        cwd=ROOT,
    )
    if appendix.get("runbook_g4_smoke_sha256") != want:
        raise SystemExit("G4 appendix SHA must match filled log sha_source_path")
    if link["filled_log_fixture"] not in appendix.get("notes", ""):
        raise SystemExit("appendix notes must reference filled log fixture name")
    print(f"OK: G4 appendix SHA ↔ filled log ({link['sha_source_path']})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()