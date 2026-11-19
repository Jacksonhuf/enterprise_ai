# 交叉表 ↔ bundle 联席入库彩排（v1.38）

团队在真签前，将 **warehouse 工单号** 交叉表与 bundle / §3 台账 / staging 工单 **同工单号** 联调。

| 状态 | Fixture |
|------|---------|
| `CROSSREF_BUNDLE_JOINT_SHAPE_ONLY` | `production-gate-crossref-bundle-joint-shape-example.json` |

校验：`validate-production-gate-crossref-bundle-joint.py`（拒绝 REHEARSAL 工单号；签核字段须含 `PLACEHOLDER`）。

与 `WAREHOUSE_TICKET_SHAPE_ONLY` 交叉表、`validate-runbook-triple-gate-rehearsal.py` 叠加；**非生产真签**。