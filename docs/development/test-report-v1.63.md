# 测试报告 v1.63（#4–#6 log、narrow 分批、CONFIRMED warehouse）

## 范围

- `schema8-production-log-rehearsal-filled-v1.63.json` + `runbook-main-table-rows4-6-log-shape-v1.63.json`
- `runbook-main-table-rows1-7-filled-log-rehearsal-v1.63.json`（#1–#7 同窗）
- `r18-narrow-open-batch-wait-v1.63.json`（5/17 batch）
- `runbook-confirmed-warehouse-bundle-rehearsal-v1.63.json`
- 彩排链 **V163**（30 步）

## 本地命令

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.63.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.63.json
```

## 结果

两条命令均退出码 0（2026-11-20 CST）。