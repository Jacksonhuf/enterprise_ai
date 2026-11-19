# 第136轮（v1.36）：implicit-default 编目、headroom 阈值门、交叉表 FILLED

## 变更摘要

- `r18-implicit-default-spot-catalog-v1.36.json`（batch-1 八 spot 显式 2% hold）。
- `r18-tolerance-tightening-v1.36.json`；`report-r18-implicit-default-spot-catalog.py` artifact。
- `validate-r18-headroom-threshold-gate.py`（min 0.002，`enforce_in_ci=false` 告警）；headroom audit 带 `threshold_alerts`。
- `production-gate-ticket-crossref-filled-shape-example.json`（`FILLED_SHAPE_ONLY`）。

## 测试

```bash
for v in v1.23 v1.24 v1.25 v1.26 v1.27 v1.28 v1.29 v1.30 v1.31 v1.33 v1.34 v1.35 v1.36; do
  python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-$v.json
done
python3 scripts/validate-r18-implicit-default-spot-catalog.py docs/development/fixtures/r18-implicit-default-spot-catalog-v1.36.json
python3 scripts/report-r18-implicit-default-spot-catalog.py /tmp/catalog.json
python3 scripts/validate-production-gate-ticket-crossref.py docs/development/fixtures/production-gate-ticket-crossref-filled-shape-example.json
python3 scripts/report-r18-tolerance-tightening-rollup.py /tmp/rollup.json
python3 scripts/validate-r18-spot-manifest-sync.py
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```