#!/usr/bin/env bash
# v1.21: mixed read-only smoke orchestration probe (not SLO bench).
set -euo pipefail
LOG="${1:-/tmp/mixed-read-probe.log}"
OUT="${2:-/tmp/mixed-read-probe-summary.json}"
ROOT="$(dirname "$0")/.."
cd "$ROOT/backend/platform"
export COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 -v ./internal/platform/api/ \
  -run 'TestR20Schema8MixedReadOnlySmokeOrchestration$' -timeout 8m 2>&1 | tee "$LOG"
python3 "$ROOT/scripts/collect-mixed-read-probe-summary.py" "$LOG" "$OUT"
echo "R09 mixed-read probe harness OK"