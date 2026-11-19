# 测试报告 v1.47（§3 联动 + 团队 staging + Figma 8 matched）

## 范围

- staging 团队票号格式彩排（CLOSED，无 PLACEHOLDER）
- `runbook-section3-defer-rollup-linkage-v1.47.json` 链式校验
- Figma checklist **8** matched（V147）

## 本地

```bash
python3 scripts/validate-staging-team-filled-rehearsal.py docs/development/fixtures/staging-alert-import-ticket-team-filled-rehearsal-v147.json
python3 scripts/validate-runbook-section3-defer-rollup-linkage.py docs/development/fixtures/runbook-section3-defer-rollup-linkage-v1.47.json
python3 scripts/validate-r02-figma-field-checklist.py docs/development/fixtures/r02-figma-field-checklist-v1.44.json
```

## 结论

runbook §3 彩排链扩展；**非**生产 staging 实填。