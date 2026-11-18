# 第95轮：混合只读烟雾编排与 R21 告警 handoff

## 变更摘要

- **`TestR20Schema8MixedReadOnlySmokeOrchestration`**：单测编排 ready + 治理四路径 + message load，36 workers 双主机。
- **`mysql-schema8-r21-alerting-handoff.md`**：签核后告警同步模板。
- **`mysql-schema8-runbook-production-fill-checklist.md`**：生产 runbook 首条实填验收清单。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.95-mixed-read-smoke-test.log`