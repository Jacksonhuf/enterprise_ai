# 测试报告 v1.67（Figma/UAT、双签、DSN dry-run）

## 增量

- `r02-figma-pixel-green-uat-rehearsal-v1.67.json` + `r02-production-uat-signoff-shape-v1.67.json`
- `runbook-sre-platform-dual-signoff-shape-v1.67.json`
- `schema8-production-ledger-dsn-dry-run-v1.67.json` + `runbook-dsn-dry-run-ledger-rehearsal-v1.67.json`
- 链 **V167**（41 步）；dashboard → `r21-open-gap-dashboard-linkage-v1.67.json`

## 本地验证

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.67.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.67.json
```