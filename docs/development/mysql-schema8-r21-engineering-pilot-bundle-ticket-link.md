# 工程彩排 bundle 与 staging 工单挂接（v1.30）

与 `staging-alert-import-ticket-closed-shape-example.json`（`CLOSED_SHAPE_ONLY`）及 `validate-staging-import-ticket.py` 对齐。

| bundle 字段 | 彩排 | 真 CONFIRMED |
|-------------|------|----------------|
| `staging_import_ticket_ref` | 空（`PENDING_TEAM_CONFIRM`）或 `STG-ALERT-REHEARSAL-SHAPE-ONLY` | staging 工单号（无 PLACEHOLDER） |
| `bundle_status` | `CONFIRMED_SHAPE_ONLY` / `APPROVED_SHAPE_ONLY` / `CONFIRMED_SIGNATORY_NAMES_SHAPE_ONLY`（v1.40） | `CONFIRMED`（双人签字 + 工单号） |

§3 变更单与 bundle 对齐（v1.31）：`mysql-schema8-r21-pilot-ledger-ticket-sync.md`。  
交叉表（v1.34）：`mysql-schema8-r21-production-gate-ticket-crossref.md`。

Agent 不代填真签或真工单闭环。形状：`engineering-pilot-bundle-*-example.json`。