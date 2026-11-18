# R02：MySQL/schema8 API 验收证据索引（增量 v0.85）

本表把计划 R02 中 **CAND-14/15/16（治理用量）** 与 **PAGE-001 消息历史** 相关的 HTTP 契约，挂到已交付的专项测试与报告。不是全量 R02 闭环。

| API（方法 + 路径） | 设计挂靠 | MySQL/schema8 证据 | 报告 |
|-------------------|----------|-------------------|------|
| `GET /api/v1/cognitive/governance/usage` | CAND-16 用量筛选分页 | `TestR07Schema8GovernanceUsageRelationalPaging`、`TestR20Schema8ParallelGovernanceUsageReads` | v0.81, v0.84 |
| `GET /api/v1/messages/{id}/load` | PAGE-001 历史投影 | `TestR07Schema8RevokedHistoryRedaction`、`TestR20Schema8ParallelMessageLoadReads` | v0.81, v0.84 |
| `GET /api/v1/sessions/continue-stream/{id}` | PAGE-001 完成重放 SSE | `TestR20Schema8DualHostContinueStreamReplay` | v0.83 |
| `PUT /api/v1/sessions/{id}` | PAGE-001 会话 CAS | `TestR20Schema8DualHostSessionRenameCAS` | v0.81 |
| `POST/DELETE .../governance/grants` | CAND-12 授权 | `TestR20Schema8DualHostConcurrentGrants` | v0.82 |
| `PUT /api/v1/cognitive/settings/{section}` | CAND-17 偏好 CAS | `TestR20Schema8DualHostSettingsProfileCAS` | v0.82 |

**双实例（R20）**：上表带 `DualHost`/`Parallel` 的用例均在共享 MySQL DSN + 双 `httptest` 主机下执行。

**未覆盖（仍属 R02 缺口）**：真实模型/A–E、视觉全态、慢查询证据、生产运维 runbook 实填。

自动化烟测：

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```