# schema8 签核就绪门（自动化 vs 人工）

v1.06：**Agent 可验证左列；右列须团队完成。**

| 自动化门（CI / 本地） | 人工门 |
|----------------------|--------|
| F-G validation 绿 | SLO 正式签核 §2 |
| `runbook-smoke-summary.json` status=pass | 工程彩排台账两行签字 |
| `scripts-verify-r18-baseline.mjs` 绿 | 生产 runbook §3 实填 |
| `validate-rehearsal-log-record.py` 可用于彩排 JSON | 生产 checklist #1–#6 |
| R18 artifact `r18-spot-screenshots` | 告警规则上线 |

彩排 JSON 样例：`docs/development/fixtures/schema8-rehearsal-log-example.json`