# R02：MySQL/schema8 API 验收证据索引（增量 v0.88）

在 [v0.87](r02-mysql-api-evidence-v0.87.md) 基础上追加：

| API | 设计挂靠 | MySQL/schema8 证据 | 报告 |
|-----|----------|-------------------|------|
| `POST /api/v1/cognitive/apps/{app_id}/release-checks` | PAGE-F2 发布前检查 | `TestR20Schema8DualHostReleaseCheckRejectsStaleRevision` | v0.88 |
| `POST /api/v1/cognitive/apps/{app_id}/releases` | PAGE-F2 发布 CAS | `TestR20Schema8DualHostReleasePublishCAS`、`mysql_application_v3_publish_test.go` | v0.88 |
| `DELETE /api/v1/cognitive/governance/grants/{id}` | CAND-12 撤权幂等 | `TestR20Schema8DualHostConcurrentGrantRevoke`、`TestR20Schema8DualHostConcurrentGrants` | v0.88 |