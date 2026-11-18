# 第114轮（v1.14）：指标接线草案、生产台账模板、build/chat 窄屏 R18

## 变更摘要

- `r21-metrics-wiring-draft.json` + 校验（6 告警 metric + tier96 扩展）。
- 生产 §3 **PENDING_TEAM_ENTRY** 台账 JSON。
- R18：**25** 抽页（`build-apps-narrow`、`chat-user-narrow`）；窄屏 CI 容差见 `max-diff-ratio.json`。

## 测试

```bash
python scripts/validate-r21-metrics-wiring-draft.py docs/development/fixtures/r21-metrics-wiring-draft.json
python scripts/validate-production-ledger.py docs/development/fixtures/schema8-production-ledger-pending-example.json
```