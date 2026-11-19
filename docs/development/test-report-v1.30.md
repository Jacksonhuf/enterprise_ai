# 第130轮（v1.30）：第八轮容差、bundle 工单挂接

## 变更摘要

- `r18-tolerance-tightening-v1.30.json`（settings-tenant/models narrow；assets 保持 v1.24）。
- Bundle `staging_import_ticket_ref`；真 `CONFIRMED` 拒 PLACEHOLDER；`CONFIRMED_SHAPE_ONLY`。
- `mysql-schema8-r21-engineering-pilot-bundle-ticket-link.md`。

## 测试

```bash
for v in v1.23 v1.24 v1.25 v1.26 v1.27 v1.28 v1.29 v1.30; do
  python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-$v.json
done
python3 scripts/validate-engineering-pilot-bundle.py docs/development/fixtures/engineering-pilot-bundle-pending-example.json
python3 scripts/validate-engineering-pilot-bundle.py docs/development/fixtures/engineering-pilot-bundle-confirmed-shape-example.json
python3 scripts/validate-engineering-pilot-bundle.py docs/development/fixtures/engineering-pilot-bundle-approved-shape-example.json
python3 scripts/validate-r18-spot-manifest-sync.py
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```