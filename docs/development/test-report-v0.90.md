# 第90轮：R20 会话探测与治理 EXPLAIN 烟测

## 变更摘要

- **`TestR20Schema8DualHostSessionCreateOnSecondHost`**：在 peer `httptest` 主机上创建会话并在主 Store 可见。
- **`TestR20Schema8DualHostConcurrentSessionListReads`**：带 `app_id` 的双主机 `GET /sessions` 只读并发。
- **`TestR20Schema8DualHostCrossHostChatProbe`**：A 创建会话、B `agent-chat`、双主机消息 `load` 一致。
- **`TestR20Schema8GovernanceRelationalExplainSmoke`**：usage/exceptions 租户分页 `EXPLAIN` 非全表扫。
- 运维：`mysql-schema8-slow-query-checklist.md`；runbook §4 烟雾项更新。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.90-session-explain-test.log`