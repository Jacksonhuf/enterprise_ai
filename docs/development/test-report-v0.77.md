# 第77轮：R08 usage plan 与 DDL

## 变更摘要

- `migrations/007_usage.sql`：`cognara_fg_usage` 目标表（含 unit/capability/occurred_at 投影）与 CHECK。
- `splitUsageLayout` / `joinUsageLayout` / `verifyUsageLayout`：要求 schema7 根域；`usage` 数组拆装并保留顺序（主键 `event_id`）。
- 只读 `PlanMySQLUsage`（要求 schema7 + 关系表齐备；目标 schema8；拒绝保留表已存在）。
- `cognara-migrate-usage --check`；内部 `rehearseMySQLUsageMigration` 复制后回滚。
- CI 纳入 `cognara-migrate-usage` 构建与测试。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'Usage' \
  ./cmd/cognara-migrate-usage/
COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race ./internal/platform/applications/
```

产物：`/opt/cursor/artifacts/usage-plan-test.log`

## 未在本轮

- `ApplyMySQLUsage` / `--apply` 与 schema8 运行时
- schema8 快照 export/import