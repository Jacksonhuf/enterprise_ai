# schema8 R20 轻量并发探针基线（非正式 SLO）

验证环境 MySQL 专项测试记录的**探针规模**，供运维冒烟对照；**不是**生产 SLO 承诺。

| 探针 | 测试 | Workers | 说明 |
|------|------|---------|------|
| `/ready` | `TestR20Schema8ParallelReadyAcrossDualHosts` | 48 | 双主机；全轮 <15s（本机 httptest） |
| `continue-stream` | `TestR20Schema8ParallelContinueStreamReplay` | 32 | 双主机；重放不增 usage / 消息 |
| 治理只读 | `TestR20Schema8ParallelGovernance*Reads` | 32 | grants/usage/exceptions/audit |

正式 SLO（p95/p99、可持续 RPS）待在 R21 与团队定标后替换本表。