# R02 MySQL API 证据（v0.97）

| API | 测试 | 说明 |
|-----|------|------|
| `PUT …/settings/public` | `TestR20Schema8DualHostSettingsPublicCAS` | 双主机 CAS |
| `GET …/continue-stream`（未完成） | `TestR20Schema8DualHostContinueStreamInProgressTurn` | 409 双主机 |
| `POST …/agent-chat` + 重放 | `TestR20Schema8InProgressTurnDisconnectThenCompletedReplay` | 断连后完成轮 |

索引：[test-report-v0.97.md](test-report-v0.97.md)、[mysql-schema8-inprogress-turn-semantics.md](mysql-schema8-inprogress-turn-semantics.md)