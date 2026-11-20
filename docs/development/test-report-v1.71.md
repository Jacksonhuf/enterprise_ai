# 测试报告 v1.71

## 增量

- `runbook-real-joint-environment-connected-shape-rehearsal-v1.71.json`（5 挂钩 `CONNECTED_SHAPE`，仍 `connected_shape_only`）
- `runbook-joint-env-connected-unsigned-gate-v1.71.json`（CONNECTED shape 与未签字 apply 同窗）
- 链 **V171**（51 步）；演练表 **C51–C52**

## 本地验证（退出码 0）

```bash
python3 scripts/validate-runbook-rehearsal-chain-orchestration.py docs/development/fixtures/runbook-rehearsal-chain-orchestration-v1.71.json
python3 scripts/validate-r21-open-gap-dashboard-linkage.py docs/development/fixtures/r21-open-gap-dashboard-linkage-v1.71.json
```