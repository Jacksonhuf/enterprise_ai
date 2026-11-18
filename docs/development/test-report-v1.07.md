# 第107轮（v1.07）：R18 聊天/编辑抽页与 R21 告警 dry-run

## 变更摘要

- R18：**12** 抽页（+`chat-user`、`editor-channel`）；刷新 PNG 基线。
- R21：告警 dry-run JSON + CI 校验；说明 `mysql-schema8-r21-alerting-dry-run.md`。

## 测试

```bash
python scripts/validate-r21-alerting-dry-run.py docs/development/fixtures/r21-alerting-dry-run.rules.json
cd backend/platform && export COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke
cd frontend && npm run build && COGNARA_R18_CAPTURE=./r18-baselines COGNARA_SMOKE_BINARY=/tmp/cognara-dev npm run test:cognara-browser
npm install --no-save pixelmatch pngjs playwright-core@1.62.1
npm run verify:r18-baseline -- ./r18-baselines
```