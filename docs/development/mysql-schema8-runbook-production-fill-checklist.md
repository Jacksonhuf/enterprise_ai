# schema8 生产 runbook 首条实填验收清单

与 `mysql-schema8-production-apply-runbook.md` §3「生产实填」配合。团队完成**第一次真实生产库** apply 后，勾选下列项方可在台账登记「runbook 首条实填已验收」。

| # | 项 | 通过 |
|---|-----|------|
| 1 | §3 执行时间、DSN 库名（无密码）、变更单号已填 | ☐ |
| 2 | 七步 migrate 每步 plan SHA256 与 apply 退出码 `0` 已登记 | ☐ |
| 3 | 双实例滚动重启后各自 `GET /ready` 为 UP | ☐ |
| 4 | §4 烟雾含混合只读探针（v0.95）及 public/SSE 项（v0.96）或同等手工检查 | ☐ |
| 5 | 回退决策（备份路径或快照 SHA256）与批准人 | ☐ |
| 6 | `mysql-schema8-r21-slo-draft.md` 签核或明确「待签核」 | ☐ |
| 7 | §4 **G4** 默认门证据与 §3 变更单同工单（见 `mysql-schema8-runbook-section3-g4-linkage.md`） | ☐ |
| 8 | 三联彩排门通过或与真签等价证据（v1.33；见 `mysql-schema8-runbook-section3-triple-gate-linkage.md`） | ☐ |

**彩排对照（非生产）：**§4 默认门为 `TestR20Schema8RunbookDefaultSmoke`（v1.00）或 CI **F-G runbook default smoke**（v1.01）。步骤见 `mysql-schema8-runbook-first-fill-guide-v1.01.md`。SLO：试点 `mysql-schema8-r21-slo-pilot-signoff.md`；正式签字 `mysql-schema8-r21-slo-signoff-record.md`。

### 彩排首条勾选（验证环境 / CI，**不等同**生产台账）

| # | 项 | 彩排 |
|---|-----|------|
| A | 本地或 CI 跑通 `TestR20Schema8RunbookDefaultSmoke` | ☐ |
| B | 下载 `runbook-smoke-summary.json` 且 `status` 为 `pass` | ☐ |
| C | F-G validation 绿 + 浏览器 smoke 含 R18 抽页（v1.03+） | ☐ |
| C2 | CI artifact `r18-spot-screenshots` 已与 manifest 核对（v1.04） | ☐ |
| C3 | R18 像素基线门绿（v1.05，`scripts-verify-r18-baseline.mjs`） | ☐ |
| C4 | runbook §4 **G1–G3** 发版勾选与 release 门禁文档一致（v1.26） | ☐ |
| C5 | 彩排 **G4↔§3** 联动说明已读（v1.27；真 §3 填后方勾选主表 #7） | ☐ |
| C6 | `validate-runbook-triple-gate-rehearsal.py` 绿或 CI 等价（v1.33） | ☐ |
| C7 | `validate-runbook-section3-defer-rollup-linkage.py` 绿（v1.47；defer 汇总↔三表同窗） | ☐ |
| C8 | 首条真填演练表 JSON 绿（v1.50；`validate-runbook-first-fill-rehearsal-table.py`） | 
| D | 工程彩排台账两方确认（见 `mysql-schema8-r21-engineering-pilot-ledger-2026-11-18.md`） | ☐ |
| E | 彩排登记模板已填（非生产）见 `mysql-schema8-production-apply-rehearsal-log-template.md` | ☐ |

生产 §3 实填后，方勾选上文主表 #1–#8。