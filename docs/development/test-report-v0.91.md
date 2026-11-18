# 第91轮：schema8 会话 writer 入口与连续创建修复

## 变更摘要

- **`TryMutateSessionsV5`**：在 schema 5–8 启用（此前误限 schema 5）。
- **`createSession` HTTP**：优先走关系化 `TryMutateSessionsV5`。
- **`sessionsRelationalEqual`**：修复连续创建时误触发全量会话重写导致的 `session unavailable`。
- **测试**：`TestR20Schema8BackToBackSessionCreates`、`TestSchema5BackToBackSessionCreates`、`TestR20Schema8BackToBackSessionWriterCreates`。
- **R02**：`r02-mysql-api-evidence-v0.91.md`。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.91-session-writer-test.log`