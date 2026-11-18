# R09 压测 harness 起步（v1.11）

**非正式 SLO 基准。** 入口：

```bash
./scripts/r09-ready-probe-harness.sh
```

对应对 Go 探针 `TestR20Schema8ParallelReadyAcrossDualHosts`（48 路双实例 `/ready`）。正式 p95/p99 与签核见 `mysql-schema8-r21-slo-signoff-record.md`。