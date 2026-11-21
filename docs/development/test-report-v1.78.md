# 测试报告 v1.78

## 增量

- `r22-joint-uat-slo-signoff-crossref-ledger-v1.78.json`（3 场景↔SLO 定标目标交叉引用；`slo_signoff_crossref_ready: false`）
- `r22-joint-uat-slo-signoff-gate-rehearsal-v1.78.json`（交叉引用台账 + v1.77 备份/SLO 联合门禁同窗）
- 链 **V178**（65 步）；**C65–C66**

## 本地验证

```bash
python3 scripts/validate-r22-joint-uat-slo-signoff-gate-rehearsal.py docs/development/fixtures/r22-joint-uat-slo-signoff-gate-rehearsal-v1.78.json
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.78.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.78.json
```