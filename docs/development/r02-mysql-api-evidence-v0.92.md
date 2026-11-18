# R02：MySQL/schema8 API 验收证据索引（增量 v0.92）

在 [v0.91](r02-mysql-api-evidence-v0.91.md) 基础上追加：

| API | 设计挂靠 | MySQL/schema8 证据 | 报告 |
|-----|----------|-------------------|------|
| `PUT /api/v1/sessions/{id}` | PAGE-001 会话 CAS/重命名 | `TestR20Schema8DualHostSessionRenameCAS`、`TestR20Schema8DualHostSessionRenameUsesWriter` | v0.92 |
| `POST /api/v1/agent-chat/{id}` | PAGE-001 轮次持久化 | `TestR20Schema8AgentChatPersistsRelationalMessages`、`TestR20Schema8DualHostCrossHostChatProbe` | v0.92 |