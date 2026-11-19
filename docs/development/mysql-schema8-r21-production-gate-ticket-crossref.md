# 生产门变更单交叉表（v1.34）

真签前排版 bundle、§3 台账、staging 工单、exporter 附录的 **变更单 / staging 单号** 须一致。

| 字段 | 彩排值 |
|------|--------|
| `apply_change_ticket_ref` | `CHG-REHEARSAL-SHAPE-ONLY` |
| `staging_import_ticket` | `STG-ALERT-REHEARSAL-SHAPE-ONLY` |

形状：`fixtures/production-gate-ticket-crossref-shape-example.json`  
校验：`validate-production-gate-ticket-crossref.py`

与 `validate-runbook-triple-gate-rehearsal.py` 叠加；真签由团队实填。

入库模板（v1.35）：`production-gate-ticket-crossref-pending-template.json`（`PENDING_TEAM_FILL`）。