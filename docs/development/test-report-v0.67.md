# 第67轮：R07 会话 plan 与 DDL

## 变更摘要

- `migrations/004_sessions.sql`：`cognara_fg_sessions`、`cognara_fg_session_messages` 目标表与 CHECK。
- `splitSessionLayout` / `joinSessionLayout` / `verifySessionLayout`：要求 schema4 根域（拒绝 `apps`/`checks`/`grants` 权威）；sessions 为 object，messages 嵌套。
- 只读 `PlanMySQLSessions`（要求 schema4 + 应用/grant/审计表；目标 schema5；拒绝保留表已存在）。
- `cognara-migrate-sessions --check`；内部 `rehearseMySQLSessionMigration` 复制后回滚。
- CI：`cognara-migrate-sessions` 构建与测试纳入 `fg-validation`。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'Session|session' \
  ./cmd/cognara-migrate-sessions/
```

## 未在本轮

- `ApplyMySQLSessions` / CLI `--apply` 与 schema5 运行时读
- 会话/消息关系 writer 与撤权脱敏