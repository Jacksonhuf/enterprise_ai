# R02：MySQL/schema8 API 验收证据索引（增量 v0.93）

在 [v0.92](r02-mysql-api-evidence-v0.92.md) 基础上追加：

| API | 设计挂靠 | MySQL/schema8 证据 | 报告 |
|-----|----------|-------------------|------|
| `GET /api/v1/sessions/continue-stream/{id}` | PAGE-001 SSE 重放 | `TestR20Schema8DualHostContinueStreamReplay`、`TestR20Schema8ParallelContinueStreamReplay` | v0.93 |