#!/usr/bin/env bash
# v1.12: message load tier probe (32/64/96w dual-host); not production SLO.
set -euo pipefail
cd "$(dirname "$0")/../backend/platform"
export COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 -v ./internal/platform/api/ \
  -run 'TestR20Schema8TieredMessageLoadLoad$' -timeout 8m
echo "R09 message load tier harness OK"