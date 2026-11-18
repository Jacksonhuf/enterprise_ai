# 第101轮（v1.01）：CI 默认烟雾 job 与 runbook 实填指引

## 变更摘要

- **`.github/workflows/fg-runbook-default-smoke.yml`**：`workflow_dispatch` + 相关路径 push main；仅 `TestR20Schema8RunbookDefaultSmoke`。
- **`mysql-schema8-runbook-first-fill-guide-v1.01.md`**：首条实填 + 签核验收步骤。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke
```

产物：`/opt/cursor/artifacts/v1.01-ci-runbook-smoke-test.log`（本地）。合并后由 Actions **F-G runbook default smoke** 提供 CI 证据。