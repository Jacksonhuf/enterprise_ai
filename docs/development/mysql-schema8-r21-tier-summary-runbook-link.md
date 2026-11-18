# Tier 摘要 × Runbook 台账关联（v1.17）

| CI artifact | 入账用途 |
|-------------|----------|
| `runbook-smoke-summary-validation` | runbook §4 默认门读阶段 |
| **`r09-tier-probe-summary`** | 工程彩排台账 tier 行（非 SLO 签核） |
| `r18-spot-screenshots` | R18 抽页证据 |

打包模板：`fixtures/engineering-pilot-bundle-pending-example.json`（`validate-engineering-pilot-bundle.py`）。

dry-run 告警 `fg_mixed_read_smoke_high` 仍绑定 `fg.read.mixed_smoke_wall_s`；tier 摘要 **不替代** mixed-read 单测编排证据。