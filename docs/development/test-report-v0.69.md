# 第69轮：R07 schema5 session 关系 writer

## 变更摘要

- schema5 下 `UpdateOperationalContext` 将会话/消息差分写入 `cognara_fg_sessions` / `cognara_fg_session_messages`（根 `body` 不再承载 sessions）。
- `TryMutateSessionsV5` / `mutateMySQLSessionsV5`：仅允许会话域 + 审计追加的专用变异路径。
- `mutateMySQLGrantsV4` 在 schema5 上改用 `readMySQLApplicationV5` 对账。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'Session|session|Schema5'
```

## 未在本轮

- schema5 快照 export/import
- 分页列表 API 与撤权脱敏