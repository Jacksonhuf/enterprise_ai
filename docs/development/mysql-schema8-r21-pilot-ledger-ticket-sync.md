# 彩排 bundle 与 §3 台账工单号对账（v1.31）

真 **CONFIRMED** bundle 与真 **ENTERED** 台账须共用同一 `apply_change_ticket_ref`（变更单号）。

| 对象 | 字段 | 彩排 |
|------|------|------|
| `engineering-pilot-bundle-*.json` | `apply_change_ticket_ref` | `CHG-REHEARSAL-SHAPE-ONLY` |
| `schema8-production-ledger-entered-shape-example.json` | `apply_change_ticket_ref` | 同上 |
| Staging 工单 | `staging_import_ticket` | `STG-ALERT-REHEARSAL-SHAPE-ONLY`（独立 staging 导入单） |

校验：

```bash
python3 scripts/validate-pilot-ledger-bundle-ticket-sync.py \
  docs/development/fixtures/engineering-pilot-bundle-confirmed-shape-example.json \
  docs/development/fixtures/schema8-production-ledger-entered-shape-example.json
```

Agent 不代填生产变更单号。