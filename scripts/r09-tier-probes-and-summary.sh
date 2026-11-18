#!/usr/bin/env bash
# v1.16: run message-load + usage tier harnesses and write JSON summary for CI artifact.
set -euo pipefail
LOG="${1:-/tmp/r09-tier-probe.log}"
OUT="${2:-/tmp/r09-tier-probe-summary.json}"
ROOT="$(dirname "$0")/.."
: >"$LOG"
bash "$ROOT/scripts/r09-message-load-tier-harness.sh" 2>&1 | tee -a "$LOG"
bash "$ROOT/scripts/r09-usage-tier-harness.sh" 2>&1 | tee -a "$LOG"
python3 "$ROOT/scripts/collect-tier-probe-summary.py" "$LOG" "$OUT"
echo "Wrote $OUT"