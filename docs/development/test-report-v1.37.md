# 第137轮（v1.37）：batch-2 编目、headroom↔SLO、工单入库格式

## 变更摘要

- batch-2 八宽屏 spot 显式 2% hold；`r18-implicit-default-spot-catalog-v1.37.json`（implicit 清零）。
- `mysql-schema8-r21-r18-headroom-slo-linkage.md` + `validate-r18-headroom-slo-linkage.py`。
- 交叉表 `WAREHOUSE_TICKET_SHAPE_ONLY`（禁止 REHEARSAL 工单前缀）。

## 测试

```bash
for v in v1.23 v1.24 v1.25 v1.26 v1.27 v1.28 v1.29 v1.30 v1.31 v1.33 v1.34 v1.35 v1.36 v1.37; do
  python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-$v.json
done
python3 scripts/validate-r18-implicit-default-spot-catalog.py docs/development/fixtures/r18-implicit-default-spot-catalog-v1.37.json
python3 scripts/validate-r18-headroom-slo-linkage.py docs/development/fixtures/r18-headroom-slo-linkage-v1.37.json
python3 scripts/validate-production-gate-ticket-crossref.py docs/development/fixtures/production-gate-ticket-crossref-warehouse-shape-example.json
python3 scripts/report-r18-implicit-default-spot-catalog.py /tmp/catalog.json
python3 scripts/validate-r18-spot-manifest-sync.py
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```