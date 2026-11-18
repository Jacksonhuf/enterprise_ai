# 第83轮：R20 schema8 双 HTTP SSE 与就绪并发探测

## 变更摘要

- **`TestR20Schema8DualHostContinueStreamReplay`**：schema8 双主机对已完成 assistant 回合的 `continue-stream` SSE 重放一致，且不重复执行业务（usage/消息条数不变）。
- **`TestR20Schema8DualHostContinueStreamMissingMessage`**：双主机对未知 `message_id` 均返回 404。
- **`TestR20Schema8ParallelReadyAcrossDualHosts`**：48 路并发 `/ready` 在双主机间轮换（轻量探测，非 SLO 压测）。

共享夹具 `schema8DualHTTPFixture` 提取自 schema8 MySQL + 第二 `httptest` 主机。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ -run R20Schema8
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/
```

产物：`/opt/cursor/artifacts/r20-schema8-sse-ready-test.log`

## 未在本轮

- 未完成 assistant 消息 409 的 schema8 手写字段注入（文件库专项 `TestPendingMessagesCannotReplayAsCompleted` 仍覆盖语义）
- 正式压力/SLO、SSE 中途断连、依赖故障注入