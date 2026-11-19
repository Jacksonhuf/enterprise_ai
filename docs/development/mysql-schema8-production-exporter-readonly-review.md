# 生产 exporter 只读上线 — 草案评审清单（v1.23）

**非自动上线。** 与 OpenMetrics mock / CI snapshot 联动验收后的团队评审用。

| # | 项 | 通过 |
|---|-----|------|
| 1 | `smoke-openmetrics-exporter-mock.sh` 绿 | ☐ |
| 2 | CI `openmetrics-ci-snapshot` 6 指标与 tier/runbook 联动 | ☐ |
| 3 | Recording rules YAML 与 wiring 一致 | ☐ |
| 4 | Scrape job **只读**、无 write 路径 | ☐ |
| 5 | staging 告警 6 规则人工导入工单（非 CLOSED 不入账） | ☐ |
| 6 | 生产 §3 台账 **ENTERED** 由平台负责人实填 | ☐ |

Agent 不代签。彩排 bundle 仍为 `PENDING_TEAM_CONFIRM` / PLACEHOLDER `CONFIRMED` 形状。

签字形状样例（v1.24）：`fixtures/production-exporter-review-signed-shape-example.json`（`validate-production-exporter-review.py`）。

APPROVED 形状（v1.25）：`production-exporter-review-approved-shape-example.json`。