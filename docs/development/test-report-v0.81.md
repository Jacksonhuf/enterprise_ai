# 第81轮：R20 schema8 双 HTTP + R07 治理分页/撤权脱敏

## 变更摘要

- **`PrepareSchema8IntegrationDB`**：从种子 DSN 一键升到 schema8（与迁移/快照夹具一致），供 API 与并发测试复用。
- **R20 `TestR20Schema8DualHostSessionRenameCAS`**：两个 `httptest` API 主机、共享 MySQL，并发会话标题 CAS（1×200、1×409）。
- **R07 `TestR07Schema8GovernanceUsageRelationalPaging`**：`governance/usage` 关系化分页与游标无重复 `event_id`。
- **R07 `TestR07Schema8RevokedHistoryRedaction`**：撤销 `grant-service` 后消息历史脱敏（`content_redacted`、无 `result`）。
- **运行时修复（schema8）**：
  - 根写入器在 schema ≥5 时持久化会话增量（此前仅 v5）。
  - `TryMutateGrantsV4` 在 schema 6–8 启用；grant writer 读取/校验 v8。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation'
export COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/ -run 'PrepareSchema8|R20Schema8|V8Snapshot'
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/
```

产物：`/opt/cursor/artifacts/r20-r07-schema8-http-test.log`

## 未在本轮

- R20 压力/SSE 中断与 SLO 对照
- grant/settings 全量 HTTP 并发矩阵
- 生产运维 apply 登记