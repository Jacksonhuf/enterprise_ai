# 第74轮：R08 exceptions plan 与 DDL

## 变更摘要

- `migrations/006_exceptions.sql`：`cognara_fg_exceptions` 目标表（含 ordinal/status/request_id 投影）与 CHECK。
- `splitExceptionLayout` / `joinExceptionLayout` / `verifyExceptionLayout`：要求 schema6 根域；`exceptions` 数组拆装并保留顺序。
- 只读 `PlanMySQLExceptions`（要求 schema6 + 关系表齐备；目标 schema7；拒绝保留表已存在）。
- `cognara-migrate-exceptions --check`；内部 `rehearseMySQLExceptionMigration` 复制后回滚。
- `sql_store`：schema6 经 `updateMySQLV3RootTx` 写根域（如追加 exceptions）。
- CI 纳入 `cognara-migrate-exceptions` 构建与测试。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'Exception' \
  ./cmd/cognara-migrate-exceptions/
```

## 未在本轮

- `ApplyMySQLExceptions` / `--apply` 与 schema7 运行时
- usage 关系域