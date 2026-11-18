# MySQL schema8 慢查询核对清单（0.1）

与 `TestR20Schema8GovernanceRelationalExplainSmoke` 及治理关系化读路径对齐。生产在代表数据量下复核。

## 治理分页（tenant 过滤）

| 表 | 典型查询 | 期望 |
|----|----------|------|
| `cognara_fg_usage` | `WHERE tenant_id=? ORDER BY ordinal DESC LIMIT n` | `EXPLAIN.type` 非 `ALL`；`key` 含 `usage_tenant_*` 或 PK 前缀 |
| `cognara_fg_exceptions` | 同上 | `key` 含异常表 tenant 复合索引 |

自动化：

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -count=1 ./internal/platform/applications/ -run TestR20Schema8GovernanceRelationalExplainSmoke
```

## 登记（运维）

| 字段 | 说明 |
|------|------|
| 采样时间 | |
| 行数（usage/exceptions） | |
| `EXPLAIN` 截图或 JSON | 附于变更单 |
| p95 观测 | 待 R21 定标后与 `/cognitive/governance/*` 对齐 |