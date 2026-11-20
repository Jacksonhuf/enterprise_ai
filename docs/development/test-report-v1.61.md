# 测试报告 v1.61（主表签字路径、narrow 盘点、G4 staging 三联）

## 范围

- `runbook-main-table-signoff-path-rehearsal-v1.61.json`（主表 #1–#8 真签字路径彩排）
- `r18-narrow-only-defer-inventory-v1.61.json`（20 narrow / 3 defer / 17 open）
- `runbook-g4-staging-ticket-triple-sync-v1.61.json`
- 彩排链 **V161**（24 步）、dashboard **V161**

## 本地命令

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.61.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.61.json
```

## 结果

两条命令均退出码 0（2026-11-20 CST）。