#!/usr/bin/env bash
# v1.13: governance usage tier probe (32/64/96w); not production SLO.
set -euo pipefail
cd "$(dirname "$0")/../backend/platform"
export COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ \
  -run 'TestR20Schema8TieredGovernanceUsageLoad$' -timeout 8m
echo "R09 usage tier harness OK"