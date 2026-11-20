# 测试报告 v1.73

## 增量

- `runbook-r20-pressure-sse-fault-measurement-shape-v1.73.json`（4 探针，映射现有 R20 Go 测试名）
- `runbook-r20-s5-capacity-gate-rehearsal-v1.73.json`
- 链 **V173**（55 步）；**C55–C56**

## 本地验证

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.73.json
```