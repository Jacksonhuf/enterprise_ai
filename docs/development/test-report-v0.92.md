# 第92轮：会话 rename 与 agent-chat 关系化写入

## 变更摘要

- **`applySessionMutation`**：`createSession`、`renameSession`、`agent-chat` 持久化统一优先 `TryMutateSessionsV5`。
- **`TestR20Schema8AgentChatPersistsRelationalMessages`**：双轮对话、4 条关系化消息行。
- **`TestR20Schema8DualHostSessionRenameUsesWriter`**：chat 后 peer 主机 `PUT /sessions/{id}` 重命名。
- **R02**：`r02-mysql-api-evidence-v0.92.md`。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.92-session-mutate-test.log`