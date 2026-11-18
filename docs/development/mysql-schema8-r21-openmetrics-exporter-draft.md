# OpenMetrics exporter 样例（v1.16）

**非生产。** 样例正文：`fixtures/openmetrics-exporter-draft.txt`；`expr` 名与 `r21-recording-rules-draft.yaml` 一致。

建议 scrape：`job=cognara-fg-ci-exporter`，`metrics_path=/metrics`。数值来自试点观测占位，**非** live 进程输出。

Tier 实测摘要见 CI artifact **`r09-tier-probe-summary`**（`collect-tier-probe-summary.py`）。