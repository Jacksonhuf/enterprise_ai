# staging 告警 import 工单闭环（模板）

与 `mysql-schema8-r21-staging-import-acceptance.md` 配合。**Agent 不代填 CLOSED。**

| 阶段 | 工单状态 | 团队动作 |
|------|----------|----------|
| CI 绿 | — | 下载 `r21-staging-alert-render` |
| 导入 staging | `PENDING_SRE_CLOSE` | 见 `fixtures/staging-alert-import-ticket-example.json` |
| 验收完成 | `CLOSED` | 填 `imported_at_cst`、`verified_by_sre`；`validate-staging-import-ticket.py` 通过 |
| 形状彩排 | `CLOSED_SHAPE_ONLY` | `fixtures/staging-alert-import-ticket-closed-shape-example.json`（**非生产**，v1.29） |

生产 SLO 签核与 §3 真仓入账仍走 `mysql-schema8-r21-team-signoff-request.md`。