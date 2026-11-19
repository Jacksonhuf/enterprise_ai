# OpenMetrics exporter 只读 mock（v1.18）

```bash
bash scripts/smoke-openmetrics-exporter-mock.sh
```

`serve-openmetrics-exporter-mock.py` 在 loopback 暴露 `GET /metrics`。默认 `openmetrics-exporter-draft.txt`；**v1.20+** CI 使用 `COGNARA_OPENMETRICS_PATH=/tmp/openmetrics-ci-snapshot.txt`（tier + runbook 联动）。**非生产进程**。