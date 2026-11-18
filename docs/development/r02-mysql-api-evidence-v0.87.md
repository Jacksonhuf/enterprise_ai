# R02：MySQL/schema8 API 验收证据索引（增量 v0.87）

在 [v0.86](r02-mysql-api-evidence-v0.86.md) 基础上追加：

| API | 设计挂靠 | MySQL/schema8 证据 | 报告 |
|-----|----------|-------------------|------|
| `GET /api/v1/cognitive/governance/grants` | CAND-12 授权筛选分页 | `TestR20Schema8ParallelGovernanceGrantsReads`、`governance_query_test.go` | v0.87 |
| `POST /api/v1/cognitive/governance/exceptions/{id}/status` | CAND-14 异常状态 CAS | `TestR20Schema8DualHostExceptionStatusCAS`、`mysql_r20_schema8_concurrency_test.go`（存储层） | v0.87 |
| `PUT /api/v1/cognitive/apps/{app_id}` | PAGE-F2 应用草稿 CAS | `TestR20Schema8DualHostApplicationDraftCAS`、`TestR20Schema8DualStoreDraftCAS` | v0.87 |