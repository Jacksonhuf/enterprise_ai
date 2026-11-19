# 第129轮（v1.29）：第七轮容差、staging CLOSED_SHAPE_ONLY

## 变更摘要

- `r18-tolerance-tightening-v1.29.json`（`build-apps-narrow`；`services-narrow` 保持 0.03）。
- Staging 工单 `CLOSED_SHAPE_ONLY`；真 `CLOSED` 拒绝 PLACEHOLDER 签字段。
- Exporter 评审清单 #5 与 closure 文档对齐。

## 测试

```bash
for v in v1.23 v1.24 v1.25 v1.26 v1.27 v1.28 v1.29; do
  python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-$v.json
done
python3 scripts/validate-staging-import-ticket.py docs/development/fixtures/staging-alert-import-ticket-example.json
python3 scripts/validate-staging-import-ticket.py docs/development/fixtures/staging-alert-import-ticket-closed-shape-example.json
python3 scripts/validate-r18-spot-manifest-sync.py
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```