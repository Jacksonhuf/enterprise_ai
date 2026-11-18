#!/usr/bin/env bash
# v1.11: lightweight R09 concurrency entrypoint (not production SLO bench).
set -euo pipefail
cd "$(dirname "$0")/../backend/platform"
export COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ \
  -run 'TestR20Schema8ParallelReadyAcrossDualHosts$' -timeout 5m
echo "R09 ready probe harness OK"