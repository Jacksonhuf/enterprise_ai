# §3 实填与三联彩排门（v1.33）

生产 §3 台账 **ENTERED** 登记前，团队应已：

1. 变更单 `apply_change_ticket_ref` 在三证中一致（bundle / ledger / 可选 staging 导入单）。
2. **ENTERED** 与 exporter **APPROVED** 联席形状通过（或同等真签证据）。
3. 运行 `validate-runbook-triple-gate-rehearsal.py`（CI 已接入）；可选 `validate-production-gate-ticket-crossref.py`（v1.34 交叉表）。

彩排清单 **C6**；主表 **#8**（真 §3 实填后勾选）。与 `mysql-schema8-runbook-section3-g4-linkage.md` 叠加，不替代 G4 烟雾。

Agent 不代填真签。