# 第106轮（v1.06）：R18 抽页扩展与彩排 JSON 校验

## 变更摘要

- R18：**10** 抽页（+login、build/apps、assets）。
- `validate-rehearsal-log-record.py` + 彩排 JSON 样例（虚构数据）。
- 像素阈值评审与签核就绪门文档。

## 测试

```bash
python scripts/validate-rehearsal-log-record.py docs/development/fixtures/schema8-rehearsal-log-example.json

cd backend/platform && export COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke

cd frontend && npm run build
COGNARA_R18_CAPTURE=./r18-baselines COGNARA_SMOKE_BINARY=/tmp/cognara-dev npm run test:cognara-browser
npm install --no-save pixelmatch pngjs
npm run verify:r18-baseline -- /tmp/r18-spots
```