# Exporter 评审附录 A：§4 G4 smoke artifact（v1.28）

与 `mysql-schema8-runbook-section3-g4-linkage.md` 及生产 apply runbook **§4 G4** 对齐。团队真 **ENTERED** 仍由平台负责人在 §3 台账实填；本附录仅登记 **artifact 名 + SHA256** 形状。

| 字段 | 彩排样例 | 生产实填 |
|------|----------|----------|
| Artifact 名 | `runbook-smoke-summary-validation` | 同 CI 下载名 |
| SHA256 | `PLACEHOLDER-NOT-PRODUCTION` | 文件摘要（64 hex） |
| 关联变更单 | — | 与 §3「变更单号」同工单 |

JSON 形状：`fixtures/production-exporter-review-g4-appendix-shape-example.json`（`validate-production-exporter-review.py`）。