# 测试报告 v1.64（#5 rollback、narrow 第二批、team-fill/warehouse 同窗）

## 增量 fixture

- `schema8-production-log-rehearsal-filled-v1.64.json` + `runbook-main-table-row5-rollback-shape-v1.64.json`
- `runbook-main-table-rows4-6-log-shape-v1.64.json`、`runbook-main-table-rows1-7-filled-log-rehearsal-v1.64.json`
- `r18-narrow-open-batch-wait-v1.64.json`（第二批 6 slug，`remaining_open_after_batch: 6`）
- `runbook-teamfill-confirmed-warehouse-window-v1.64.json`
- 彩排表/对齐/链 **V164**；registry → `r21-open-gap-dashboard-linkage-v1.64.json`

## 本地验证（退出码 0）

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.64.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.64.json
```

## CI

`F-G validation`（`.github/workflows/fg-validation.yml`）已切换至 v1.64 链路与 dashboard 步骤。