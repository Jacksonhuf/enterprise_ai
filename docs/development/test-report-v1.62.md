# 测试报告 v1.62（主表 #1–#7 log、publish-review-narrow、ENTERED warehouse）

## 范围

- `runbook-main-table-rows1-7-filled-log-rehearsal-v1.62.json`（#1–#3、#7 与 filled log 同窗）
- `r18-publish-review-narrow-wait-decision-v1.62.json`
- `runbook-entered-crossref-warehouse-rehearsal-v1.62.json`
- 彩排链 **V162**（27 步）、dashboard **V162**

## 本地命令

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.62.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.62.json
```

## 结果

两条命令均退出码 0（2026-11-20 CST）。