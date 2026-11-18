# schema8 R21 staging 告警导入验收（人工）

**v1.09+**：在 SLO 签核前，SRE 可将 CI artifact `r21-staging-alert-render` 手工导入 **staging 监控**（非生产）。

| # | 项 | 验收 |
|---|-----|------|
| 1 | 下载 artifact，与 `validate-r21-staging-alert-render.py` 本地校验通过 | ☐ |
| 2 | staging 仅 `staging="true"` 规则生效 | ☐ |
| 3 | 故意拉高探针不触发生产路由 | ☐ |
| 4 | 登记工单号至 `mysql-schema8-r21-slo-signoff-record.md` §3 | ☐ |

自动化：`scripts/render-r21-alerting-staging-prometheus.py` + `validate-r21-staging-alert-render.py`。