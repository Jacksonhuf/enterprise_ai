# schema8 团队签核与生产实填 — 提请单（模板）

供平台工程与运维在 **v1.05+** 完成「台账实签 + 生产 §3 首条实填」时内部流转。**Agent 不得代签。**

## 提请事项

| # | 事项 | 责任人 | 证据 |
|---|------|--------|------|
| 1 | 工程彩排台账两行确认 | 平台工程 + SRE | `mysql-schema8-r21-engineering-pilot-ledger-2026-11-18.md` |
| 2 | R18 基线门通过 | 平台工程 | CI artifact `r18-spot-screenshots` + `scripts-verify-r18-baseline.mjs` 绿 |
| 3 | runbook 烟雾摘要 `status: pass` | SRE | `runbook-smoke-summary-validation` |
| 4 | 彩排 §3 登记（非生产库） | SRE | `mysql-schema8-production-apply-rehearsal-log-template.md` |
| 5 | SLO 正式签核 §2 | 三方 | `mysql-schema8-r21-slo-signoff-record.md` |
| 6 | 生产 runbook §3 实填 + checklist #1–6 | SRE | `mysql-schema8-production-apply-runbook.md` |

## 回复格式（复制到工单）

```
彩排确认：是/否
CI Run URL：
Artifacts：runbook-smoke-summary-validation / r18-spot-screenshots
生产 §3 计划窗口：
```