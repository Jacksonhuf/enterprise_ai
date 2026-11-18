# 第96轮：settings/public 双主机读并发与 SSE 断连恢复

## 变更摘要

- **`TestR20Schema8ParallelGovernancePublicSettingsReads`** / **`TestR20Schema8ParallelSettingsPublicSectionReads`**：32 路双主机只读。
- **`TestR20Schema8ContinueStreamEarlyDisconnectRecovery`**：客户端提前关闭后双主机重放。
- runbook §4 烟雾与 R02 v0.96；首条生产实填仍由团队 checklist 验收。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.96-public-sse-test.log`