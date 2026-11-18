# schema8 R20 轻量并发探针基线（非正式 SLO）

验证环境 MySQL 专项测试记录的**探针规模**，供运维冒烟对照；**不是**生产 SLO 承诺。

| 探针 | 测试 | Workers | 说明 |
|------|------|---------|------|
| `/ready` | `TestR20Schema8ParallelReadyAcrossDualHosts` | 48 | 双主机；全轮 <15s（本机 httptest） |
| `continue-stream` | `TestR20Schema8ParallelContinueStreamReplay` | 32 | 双主机；重放不增 usage / 消息 |
| 治理只读 | `TestR20Schema8ParallelGovernance*Reads` | 32 | grants/usage/exceptions/audit |
| 治理 `usage` 阶梯 | `TestR20Schema8TieredGovernanceUsageLoad` | 32 → 64 | 记录每档 `t.Log` 墙钟 |
| 消息 `load` 阶梯 | `TestR20Schema8TieredMessageLoadLoad` | 32 → 64 | 历史长度一致；记录墙钟 |
| 混合只读烟雾 | `TestR20Schema8MixedReadOnlySmokeOrchestration` | 36 | ready+治理+load 编排 |
| 公共配置读 | `TestR20Schema8ParallelGovernancePublicSettingsReads` 等 | 32 | governance/settings + settings/public |
| SSE 断连恢复 | `TestR20Schema8ContinueStreamEarlyDisconnectRecovery` | 1 | 部分读关闭后双主机重放 |
| settings/public 写 | `TestR20Schema8DualHostSettingsPublicCAS` | 2 | 双主机 PUT CAS |
| 进行中 turn | `TestR20Schema8DualHostContinueStreamInProgressTurn` | 2 | 409 双主机 |

正式 SLO（p95/p99、可持续 RPS）见 `mysql-schema8-r21-slo-draft.md`，团队签核后替换本表；告警映射见 `mysql-schema8-r21-alerting-handoff.md`。