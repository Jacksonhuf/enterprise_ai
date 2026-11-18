# 第100轮：runbook 默认烟雾门与 R21 正式签核记录

## 变更摘要

- **`TestR20Schema8RunbookDefaultSmoke`**：runbook 单测入口（委托写读编排）。
- **`mysql-schema8-runbook-default-smoke.md`**：默认 `go test` 命令。
- **`mysql-schema8-r21-slo-signoff-record.md`**：正式签核记录表（建议目标 + 签字栏）。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v1.00-runbook-default-smoke-test.log`