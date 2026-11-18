# R02：MySQL/schema8 API 验收证据索引（增量 v0.86）

在 [v0.85](r02-mysql-api-evidence-v0.85.md) 基础上追加：

| API | 设计挂靠 | MySQL/schema8 证据 | 报告 |
|-----|----------|-------------------|------|
| `GET /api/v1/cognitive/governance/exceptions` | CAND-14/15 异常筛选分页 | `TestR20Schema8ParallelGovernanceExceptionsReads` | v0.86 |
| `GET /api/v1/cognitive/governance/audit` | CAND-13 审计筛选分页 | `TestR20Schema8ParallelGovernanceAuditReads`、`audit_mysql_query_test.go` | v0.86 |

**缺陷修复**：schema8 治理异常列表改走 `queryMySQLExceptionsRelational`（此前仅 schema7）。