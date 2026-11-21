# 测试报告 v1.80

## 增量

- `runbook-r20-slo-team-adoption-shape-ledger-v1.80.json`（4 条采纳 shape；`team_slo_adopted: false`）
- `runbook-r20-slo-team-adoption-rehearsal-gate-v1.80.json`（采纳台账 + v1.79 dry-run 门禁同窗）
- 链 **V180**（69 步）；**C69–C70**

## 本地验证

```bash
python3 scripts/validate-runbook-r20-slo-team-adoption-rehearsal-gate.py docs/development/fixtures/runbook-r20-slo-team-adoption-rehearsal-gate-v1.80.json
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.80.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.80.json
```