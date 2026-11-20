# 测试报告 v1.66（publish-review 收口、真签模板、ENTERED 门禁）

## 增量 fixture

- `r18-narrow-open-batch-closeout-v1.66.json` / `r18-publish-review-narrow-closeout-v1.66.json`
- `runbook-main-table-production-signoff-template-v1.66.json`（主表 #1–#8）
- `runbook-entered-ledger-write-gate-v1.66.json`
- 彩排表/对齐/链 **V166**（38 步）；registry → `r21-open-gap-dashboard-linkage-v1.66.json`

## 本地验证（退出码 0）

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.66.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.66.json
```