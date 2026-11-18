# schema8 工程彩排台账（2026-11-18，非正式签核）

**本页记录验证环境自动化证据，不替代 `mysql-schema8-r21-slo-signoff-record.md` §2 正式签字。**

| 项 | 值 |
|----|-----|
| 默认门单测 | `TestR20Schema8RunbookDefaultSmoke` |
| CI workflow | F-G runbook default smoke；F-G validation 摘要 artifact |
| 浏览器 R18 抽页 | `frontend/scripts-smoke.mjs`（v1.03+） |
| R18 截图 artifact | `r18-spot-screenshots`（v1.04+，validation） |
| 摘要 artifact 名 | `runbook-default-smoke` / `runbook-smoke-summary-validation` |
| §3 彩排登记模板 | `mysql-schema8-production-apply-rehearsal-log-template.md` |
| 团队提请单 | `mysql-schema8-r21-team-signoff-request.md` |
| R18 像素门 | `scripts-verify-r18-baseline.mjs`（v1.05） |

## 待团队填入（正式签核前移）

| 角色 | 彩排确认（是/否） | 日期 | 工单或签字 |
|------|-------------------|------|------------|
| 平台工程负责人 | | | |
| 运维/SRE 负责人 | | | |

### 彩排确认最小集（v1.04）

1. F-G validation 绿，下载 `r18-spot-screenshots`（7 张 + `manifest.json`）。
2. `runbook-smoke-summary.json` 中 `status: pass`。
3. 在彩排登记模板填写 §3 字段（非生产库名）。
4. 上表两行签字或工单号。

彩排通过后，将 Run URL 或 artifact 下载链接粘贴至 `mysql-schema8-r21-slo-signoff-record.md` §3。