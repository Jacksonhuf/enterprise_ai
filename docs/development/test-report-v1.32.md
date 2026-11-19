# 第132轮（v1.32）：容差 rollup、三证工单、联席形状

## 变更摘要

- `report-r18-tolerance-tightening-rollup.py` → artifact `r18-tolerance-tightening-rollup`。
- `validate-pilot-ticket-triple-sync.py`；`validate-production-gate-joint-shape.py`。
- `production-gate-entered-exporter-joint-shape-example.json`。

## 测试

```bash
python3 scripts/report-r18-tolerance-tightening-rollup.py /tmp/r18-tolerance-tightening-rollup-v1.32.json
python3 scripts/validate-pilot-ticket-triple-sync.py \
  docs/development/fixtures/engineering-pilot-bundle-approved-shape-example.json \
  docs/development/fixtures/schema8-production-ledger-entered-shape-example.json \
  docs/development/fixtures/staging-alert-import-ticket-closed-shape-example.json
python3 scripts/validate-production-gate-joint-shape.py \
  docs/development/fixtures/production-gate-entered-exporter-joint-shape-example.json
for v in v1.23 v1.24 v1.25 v1.26 v1.27 v1.28 v1.29 v1.30 v1.31; do
  python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-$v.json
done
python3 scripts/validate-r18-spot-manifest-sync.py
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```