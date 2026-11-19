#!/usr/bin/env python3
"""Validate engineering pilot rehearsal bundle JSON (v1.17+)."""
import json
import sys
from pathlib import Path

REQUIRED = frozenset(
    {
        "bundle_status",
        "runbook_smoke_artifact",
        "tier_probe_artifact",
        "r18_screenshot_artifact",
        "staging_alert_artifact",
        "openmetrics_ci_snapshot_artifact",
        "openmetrics_linked_metric_count",
        "linked_runbook_checklist",
        "confirmed_by_platform",
        "confirmed_by_sre",
        "staging_import_ticket_ref",
        "apply_change_ticket_ref",
        "notes",
    }
)


def main() -> None:
    path = Path(sys.argv[1])
    data = json.loads(path.read_text())
    if set(data.keys()) != REQUIRED:
        raise SystemExit(f"keys must be {sorted(REQUIRED)}")
    if data["tier_probe_artifact"] != "r09-tier-probe-summary":
        raise SystemExit("tier_probe_artifact must be r09-tier-probe-summary")
    if data["openmetrics_ci_snapshot_artifact"] != "openmetrics-ci-snapshot":
        raise SystemExit("openmetrics_ci_snapshot_artifact must be openmetrics-ci-snapshot")
    if data["openmetrics_linked_metric_count"] != 6:
        raise SystemExit("openmetrics_linked_metric_count must be 6")
    ref = data["staging_import_ticket_ref"]
    if data["bundle_status"] == "CONFIRMED":
        if not data["confirmed_by_platform"] or not data["confirmed_by_sre"]:
            raise SystemExit("CONFIRMED bundle requires both confirmations")
        for k in ("confirmed_by_platform", "confirmed_by_sre"):
            if "PLACEHOLDER" in data[k]:
                raise SystemExit(f"CONFIRMED bundle must not use PLACEHOLDER in {k}")
        if not ref or "PLACEHOLDER" in ref:
            raise SystemExit("CONFIRMED bundle requires staging_import_ticket_ref")
        if not data["apply_change_ticket_ref"] or "PLACEHOLDER" in data["apply_change_ticket_ref"]:
            raise SystemExit("CONFIRMED bundle requires apply_change_ticket_ref")
    elif data["bundle_status"] == "CONFIRMED_SIGNATORY_NAMES_SHAPE_ONLY":
        for k in ("confirmed_by_platform", "confirmed_by_sre"):
            val = data[k]
            if "PLACEHOLDER" in val:
                raise SystemExit(f"{k} must not use PLACEHOLDER for signatory names shape")
            if "SIGNATORY" not in val or "SHAPE" not in val:
                raise SystemExit(f"{k} must be SIGNATORY-*-SHAPE-* for names shape")
        if "REHEARSAL" in data["apply_change_ticket_ref"].upper():
            raise SystemExit("signatory names shape expects warehouse ticket refs")
    elif data["bundle_status"] in ("APPROVED_SHAPE_ONLY", "CONFIRMED_SHAPE_ONLY"):
        pass
    elif data["bundle_status"] != "PENDING_TEAM_CONFIRM":
        raise SystemExit(
            "bundle_status must be PENDING_TEAM_CONFIRM, CONFIRMED, "
            "CONFIRMED_SIGNATORY_NAMES_SHAPE_ONLY, "
            "CONFIRMED_SHAPE_ONLY, or APPROVED_SHAPE_ONLY"
        )
    print(f"OK: engineering pilot bundle ({data['bundle_status']})")
    print(f"Validated {path}")


if __name__ == "__main__":
    main()