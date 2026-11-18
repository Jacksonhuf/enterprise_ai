# Prom 指标接线草案（v1.14）

**不自动部署。** 将 dry-run 告警中的 `metric` 名与 CI/探针产出对齐，供 SRE 手工接线。

| 产出 | 文件 |
|------|------|
| 接线表 | `fixtures/r21-metrics-wiring-draft.json` |
| 告警源 | `fixtures/r21-alerting-dry-run.rules.json` |
| 校验 | `validate-r21-metrics-wiring-draft.py` |

生产 §3 真值入账仍走 `fixtures/schema8-production-ledger-pending-example.json`（**Agent 不代填 ENTERED**）。