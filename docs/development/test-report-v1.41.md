# 第141轮（v1.41）：chat-user、子集扩展、CONFIRMED 彩排

## 变更摘要

- **无** `max-diff-ratio` 变更（chat-user v1.40 hold）；`build-apps-narrow` v1.29 defer。
- Headroom subset 增加 `login-public`（v1.41 fixture）。
- `engineering-pilot-bundle-confirmed-warehouse-rehearsal-example.json`（`bundle_status: CONFIRMED`）。

## 测试

```bash
python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-v1.41.json
python3 scripts/validate-engineering-pilot-bundle-confirmed-warehouse-rehearsal.py docs/development/fixtures/engineering-pilot-bundle-confirmed-warehouse-rehearsal-example.json
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```