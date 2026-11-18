# 第86轮：R20 治理 audit/exceptions 并发读 + schema8 异常列表修复

## 变更摘要

- **`TestR20Schema8ParallelGovernanceExceptionsReads`** / **`TestR20Schema8ParallelGovernanceAuditReads`**：双 HTTP 主机 32 路只读并发。
- **修复**：`queryMySQLGovernanceArray` 在 schema8 对 `exceptions` 使用关系化查询（与 schema7 对齐）。
- **运维**：runbook §3 增加登记表填写样例。
- **R02**：`r02-mysql-api-evidence-v0.86.md` 挂接 audit/exceptions API。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run R20Schema8
```

产物：`/opt/cursor/artifacts/v0.86-r20-governance-read-test.log`