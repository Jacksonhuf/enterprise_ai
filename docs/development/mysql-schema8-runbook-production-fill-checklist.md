# schema8 生产 runbook 首条实填验收清单

与 `mysql-schema8-production-apply-runbook.md` §3「生产实填」配合。团队完成**第一次真实生产库** apply 后，勾选下列项方可在台账登记「runbook 首条实填已验收」。

| # | 项 | 通过 |
|---|-----|------|
| 1 | §3 执行时间、DSN 库名（无密码）、变更单号已填 | ☐ |
| 2 | 七步 migrate 每步 plan SHA256 与 apply 退出码 `0` 已登记 | ☐ |
| 3 | 双实例滚动重启后各自 `GET /ready` 为 UP | ☐ |
| 4 | §4 烟雾含混合只读探针（v0.95）或同等手工检查 | ☐ |
| 5 | 回退决策（备份路径或快照 SHA256）与批准人 | ☐ |
| 6 | `mysql-schema8-r21-slo-draft.md` 签核或明确「待签核」 | ☐ |

**彩排对照（非生产）：**验证环境可使用 test-report-v0.95 中的 `TestR20Schema8MixedReadOnlySmokeOrchestration` 作为 §4 第 4 项的自动化替身。