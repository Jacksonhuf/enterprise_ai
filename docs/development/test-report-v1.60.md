# 测试报告 v1.60（主表 #8、G4 SHA、groups/roles）

## 范围

- `runbook-main-table8-dashboard-window-check-v1.60.json` ↔ triple-gate registry V160
- `runbook-g4-filled-log-sha-linkage-v1.60.json` + `production-exporter-review-g4-appendix-rehearsal-v1.60.json`
- `r18-groups-roles-tab-defer-decision-v1.60.json`（CAND-09/11）
- 彩排链 **V160**（21 步）、dashboard **V160**

## 本地命令

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.60.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.60.json
```

## 结果

两条命令均退出码 0（2026-11-20 CST）。