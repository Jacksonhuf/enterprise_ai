# ENTERED 台账与 CONFIRMED bundle 同工单同桌（v1.42）

团队在真签前校验 §3 **ENTERED** 台账与 **CONFIRMED** 彩排 bundle 共用变更单 / staging 工单号。

| 状态 | Fixture |
|------|---------|
| `ENTERED_CONFIRMED_BUNDLE_JOINT_SHAPE_ONLY` | `production-gate-entered-confirmed-bundle-joint-shape-example.json` |

校验：`validate-production-gate-entered-confirmed-bundle-joint.py`（非 `PLACEHOLDER` 工单；签核仍彩排）。

与 `validate-production-gate-crossref-bundle-joint.py`、triple-gate runbook 叠加。