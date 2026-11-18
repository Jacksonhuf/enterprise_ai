# schema8 R21 告警 dry-run（v1.07）

**不部署到生产**；签核前用于 SRE 与试点表对照。

| 文件 | 说明 |
|------|------|
| `fixtures/r21-alerting-dry-run.rules.json` | 试点阈值 ×~1.5 的工程建议门 |
| `scripts/validate-r21-alerting-dry-run.py` | CI 结构校验 |

签核后：将规则抄入监控系统，并在 `mysql-schema8-r21-alerting-handoff.md` 登记规则 ID。

**Staging 渲染（v1.08）：** `scripts/render-r21-alerting-staging-prometheus.py`；CI artifact `r21-staging-alert-render`（**不自动 apply**）。