# R02 MySQL API 证据（v0.96）

| API | 测试 | 说明 |
|-----|------|------|
| `GET …/governance/settings` | `TestR20Schema8ParallelGovernancePublicSettingsReads` | R01 公告列表并发 |
| `GET …/settings/public` | `TestR20Schema8ParallelSettingsPublicSectionReads` | 公共配置段并发读 |
| `GET …/continue-stream/{id}` | `TestR20Schema8ContinueStreamEarlyDisconnectRecovery` | 断连后重放 |

索引：[test-report-v0.96.md](test-report-v0.96.md)