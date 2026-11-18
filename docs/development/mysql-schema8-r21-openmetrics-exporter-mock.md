# OpenMetrics exporter 只读 mock（v1.18）

```bash
bash scripts/smoke-openmetrics-exporter-mock.sh
```

`serve-openmetrics-exporter-mock.py` 在 loopback 暴露 `GET /metrics`，正文来自 `openmetrics-exporter-draft.txt`。**非生产进程**，仅供接线彩排与 CI smoke。