# 生产 §3 台账首条真填模板（v1.48）

**JSON 模板**：`fixtures/schema8-production-ledger-first-production-fill-template-v1.48.json`  
**交叉表**：`fixtures/production-gate-ticket-crossref-team-fill-staging-v148.json`（team-fill staging）

校验：`validate-schema8-production-ledger-first-fill-template.py`

团队在生产 apply 完成后，将 `PENDING_TEAM_ENTRY` 填为 `ENTERED` 真值；Agent 不代签。彩排 crossref 仅保证工单与 staging 票格式一致。