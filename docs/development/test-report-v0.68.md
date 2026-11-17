# 第68轮：R07 session apply 与 schema5 读

## 变更摘要

- `ApplyMySQLSessions` + `cognara-migrate-sessions --apply --expect-sha256`：schema4 单事务拆 sessions/messages 并切换 schema5。
- `readMySQLApplicationV5`、`OpenMySQL` schema5 启动校验；`readMySQLApplicationRelational` 路由 v5。
- 根域 writer：schema5 禁止经 body 改 sessions（与 grants 对称）；schema5 仍可用 `TryMutateGrantsV4`。
- `PlanMySQLSessions` 登记 `apply_supported: true`。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'Session|session' \
  ./cmd/cognara-migrate-sessions/
```

## 未在本轮

- schema5 快照 export/import
- 会话/消息关系 writer 与分页查询