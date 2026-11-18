# R02：MySQL/schema8 API 验收证据索引（增量 v0.90）

在 [v0.89](r02-mysql-api-evidence-v0.89.md) 基础上追加：

| API | 设计挂靠 | MySQL/schema8 证据 | 报告 |
|-----|----------|-------------------|------|
| `POST …/apps/{id}/sessions` | PAGE-001 会话创建 | `TestR20Schema8DualHostSessionCreateOnSecondHost`、`TestR20Schema8DualHostCrossHostChatProbe` | v0.90 |
| `GET /api/v1/sessions` | PAGE-001 会话列表 | `TestR20Schema8DualHostConcurrentSessionListReads` | v0.90 |
| `POST /api/v1/agent-chat/{id}` + `GET …/messages/load` | PAGE-001 跨主机续聊 | `TestR20Schema8DualHostCrossHostChatProbe` | v0.90 |
| 治理关系化读（内部） | CAND-16/14 慢查询准备 | `TestR20Schema8GovernanceRelationalExplainSmoke`、`mysql-schema8-slow-query-checklist.md` | v0.90 |