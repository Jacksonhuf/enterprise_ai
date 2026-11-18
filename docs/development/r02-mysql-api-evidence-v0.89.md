# R02：MySQL/schema8 API 验收证据索引（增量 v0.89）

在 [v0.88](r02-mysql-api-evidence-v0.88.md) 基础上追加：

| API | 设计挂靠 | MySQL/schema8 证据 | 报告 |
|-----|----------|-------------------|------|
| `POST /api/v1/cognitive/apps`（TEMPLATE） | PAGE-F2 模板创建 CAS | `TestR20Schema8DualHostTemplateCopyRejectsStaleRevision`、`TestR20Schema8DualHostTemplateCopyConcurrentFresh` | v0.89 |
| `POST/DELETE …/governance/grants` | CAND-12 授撤交叉 | `TestR20Schema8DualHostGrantRevokePostRace`、`TestR20Schema8DualHostConcurrentGrants` | v0.89 |