# 测试报告 v1.75

## 增量

- `runbook-r21-alert-dryrun-shape-ledger-v1.75.json`（3 条告警 dry-run，`dryrun_fired: false`）
- `runbook-r21-oncall-handoff-gate-rehearsal-v1.75.json`（主/备 on-call PLACEHOLDER）
- 链 **V175**（59 步）；**C59–C60**

## 本地验证

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.75.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.75.json
```