# schema8 生产 apply 彩排登记（**非生产**，首条 §3 实填前）

与 `mysql-schema8-production-apply-runbook.md` §3 字段对齐。仅在**彩排库**或验证环境填写；**禁止**将本表当作生产台账。

| 字段 | 彩排值（团队填写） |
|------|-------------------|
| 执行时间（北京时间） | |
| DSN 库名（无密码） | |
| 变更单号 | `REHEARSAL-` |
| 每步 plan SHA256 | |
| apply 退出码 | |
| 双实例 `/ready` | |
| runbook 烟雾 artifact | `runbook-default-smoke` 或本地 summary JSON |
| R18 抽页 artifact | `r18-spot-screenshots`（v1.04+） |
| 工程彩排台账确认 | 见 `mysql-schema8-r21-engineering-pilot-ledger-2026-11-18.md` |

彩排完成后：将 Run URL + artifact 名录入台账；再在 **生产** runbook §3「生产值」表实填。