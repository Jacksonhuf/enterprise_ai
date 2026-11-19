# 第127轮（v1.27）：第五轮容差、§3↔G4 联动

## 变更摘要

- `r18-tolerance-tightening-v1.27.json`（`publish-review-narrow`、`session-sidebar-narrow`）。
- `mysql-schema8-runbook-section3-g4-linkage.md`；§3 表 G4 证据列；清单 **#7** / 彩排 **C5**。

## 测试

```bash
for v in v1.23 v1.24 v1.25 v1.26 v1.27; do
  python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-$v.json
done
python3 scripts/validate-r18-spot-manifest-sync.py
python3 scripts/validate-production-ledger.py docs/development/fixtures/schema8-production-ledger-entered-shape-example.json
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```