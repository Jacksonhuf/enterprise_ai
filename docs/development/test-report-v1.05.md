# 第105轮（v1.05）：R18 像素基线门与签核提请单

## 变更摘要

- **`frontend/r18-baselines/`**：7 张 PNG + manifest。
- **`frontend/scripts-verify-r18-baseline.mjs`**：pixelmatch 门（默认 max 2%）。
- **CI**：validation 在浏览器 smoke 后跑基线对比。
- **`mysql-schema8-r21-team-signoff-request.md`**：台账/生产实填提请模板。

## 测试

```bash
cd backend/platform && export COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke

cd frontend && npm run build
mkdir -p /tmp/r18-spots
COGNARA_R18_CAPTURE=/tmp/r18-spots COGNARA_SMOKE_BINARY=/tmp/cognara-dev npm run test:cognara-browser
npm install --no-save pixelmatch pngjs
npm run verify:r18-baseline -- /tmp/r18-spots
```