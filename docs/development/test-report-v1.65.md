# 测试报告 v1.65（narrow 收尾、#8 签路径、ENTERED 入库）

## 增量 fixture

- `r18-narrow-open-batch-wait-v1.65.json`（第三批 5 slug，`remaining_open_after_batch: 1`）
- `runbook-main-table8-dashboard-window-check-v1.65.json` + `runbook-main-table-signoff-path-rehearsal-v1.65.json`
- `runbook-main-table8-signoff-dashboard-path-v1.65.json`、`r21-dashboard-triple-gate-window-registry-v1.65.json`
- `runbook-section3-entered-warehouse-intake-v1.65.json`
- 彩排表/对齐/链 **V165**（34 步）；registry → `r21-open-gap-dashboard-linkage-v1.65.json`

## 本地验证（退出码 0）

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.65.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.65.json
```

## CI

`F-G validation` 已切换至 v1.65 链路与 dashboard 步骤。