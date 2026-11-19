# Runbook §3 生产实填与 §4 G4 联动（v1.27）

团队完成 `mysql-schema8-production-apply-runbook.md` **§3** 生产登记后，方可登记台账 **ENTERED**（形状见 `fixtures/schema8-production-ledger-entered-shape-example.json`，**Agent 不代填真值**）。

## 联动顺序

1. **§3** 填执行时间、DSN（无密码）、七步 plan/apply SHA256、变更单号（见 §3 表）。
2. 在隔离或生产 DSN 上跑通 **§4 G4** 默认门（`TestR20Schema8RunbookDefaultSmoke` 或 CI artifact `runbook-smoke-summary-validation` 且 `status=pass`）。
3. 勾选 `mysql-schema8-runbook-production-fill-checklist.md` 主表 **#1–#6** 与 **#7（G4 联动）**。
4. 留存 smoke summary 的 artifact 名/SHA256 与 §3 变更单同一工单。

## 彩排（非生产）

勾选彩排 **A + B + C5**：A/B 为默认门与 summary；C5 声明 G4 证据将随 §3 实填一并归档（仍为 PLACEHOLDER 彩排）。

## Release 门禁

合入 `main` 前仍须满足 `mysql-schema8-r20-release-gate-r18-manifest.md`（R18 38 面 + 容差 v1.23–v1.27）。