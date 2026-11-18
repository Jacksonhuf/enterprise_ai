#!/usr/bin/env bash
# v1.18: smoke read-only /metrics mock (CI-safe, loopback only).
set -euo pipefail
ROOT="$(dirname "$0")/.."
PORT=19091
python3 "$ROOT/scripts/serve-openmetrics-exporter-mock.py" "$PORT" &
PID=$!
trap 'kill "$PID" 2>/dev/null || true' EXIT
for _ in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:${PORT}/metrics" -o /tmp/om-mock.txt; then
    break
  fi
  sleep 0.2
done
test -s /tmp/om-mock.txt
python3 "$ROOT/scripts/validate-openmetrics-exporter-draft.py" /tmp/om-mock.txt
echo "OpenMetrics exporter mock smoke OK"