# 第85轮：R20 聊天取消 + 快照演练 + R02 API 证据索引

## 变更摘要

- **R20 `TestR20Schema8AgentChatCanceledContextBeforePersist` / `ExpiresDuringUpstreamCall`**：已取消或极短 deadline 的 agent-chat 不在 schema8 上残留半写入消息。
- **R20 `TestR20Schema8RunbookSnapshotImportDrill`**：Prepare → export → 隔离库 import → 双连接 usage 一致（运维回退彩排）。
- **运维**：`mysql-schema8-production-apply-runbook.md` 增加 §5 快照回退演练表。
- **R02**：`r02-mysql-api-evidence-v0.85.md` 治理用量/消息历史/双实例 API 与测试映射。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/ -run R20Schema8RunbookSnapshotImportDrill
go test -p 1 -race -count=1 ./internal/platform/api/ -run R20Schema8
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/
```

产物：`/opt/cursor/artifacts/v0.85-r20-r02-test.log`

## 未在本轮

- 流式传输中途 TCP 断连（当前 agent-chat 在完成计算后一次性写 SSE）
- R02 全量页面截图与字段核对