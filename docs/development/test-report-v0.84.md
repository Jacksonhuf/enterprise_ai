# 第84轮：R20 治理/消息并发读 + schema8 apply 运维模板

## 变更摘要

- **`TestR20Schema8ParallelGovernanceUsageReads`**：32 路并发只读 `governance/usage` 分页，双 HTTP 主机轮换。
- **`TestR20Schema8ParallelMessageLoadReads`**：完成一轮 agent-chat 后，24 路并发 `messages/load`，双主机结果条数一致（user+assistant）。
- **`mysql-schema8-production-apply-runbook.md`**：生产 plan→apply 梯登记模板（对齐 `PrepareSchema8IntegrationDB` 顺序与 R20 烟雾项）。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ -run R20Schema8
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/
```

产物：`/opt/cursor/artifacts/r20-schema8-read-concurrency-test.log`

## 未在本轮

- 正式压力与 SLO
- 生产环境真实 apply 填表（模板已就绪，待运维执行）