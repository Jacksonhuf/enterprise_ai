# 第104轮（v1.04）：R18 截图基线与抽页扩展

## 变更摘要

- **浏览器**：R18 抽页扩展至 7 面（含拒绝态 `forbidden-user`）；`COGNARA_R18_CAPTURE` 输出 PNG + `manifest.json`。
- **CI**：validation 上传 artifact **`r18-spot-screenshots`**。
- **运维**：§3 彩排登记模板；台账确认最小集。

## 测试

```bash
cd backend/platform && export COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke

mkdir -p /tmp/r18-spots
cd frontend && COGNARA_R18_CAPTURE=/tmp/r18-spots COGNARA_SMOKE_BINARY=/tmp/cognara-dev npm run test:cognara-browser
ls /tmp/r18-spots
```