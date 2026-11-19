# 第133轮（v1.33）：grants 显式容差、三联 runbook 门

## 变更摘要

- `r18-tolerance-tightening-v1.33.json`（`governance-grants-narrow` 0.021）。
- `validate-runbook-triple-gate-rehearsal.py`；清单 **#8** / **C6**；rollup 收口 v1.33。

## 测试

```bash
for v in v1.23 v1.24 v1.25 v1.26 v1.27 v1.28 v1.29 v1.30 v1.31 v1.33; do
  python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-$v.json
done
python3 scripts/validate-runbook-triple-gate-rehearsal.py
python3 scripts/validate-r18-spot-manifest-sync.py
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```