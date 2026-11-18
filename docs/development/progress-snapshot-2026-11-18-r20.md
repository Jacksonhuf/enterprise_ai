# F/G 进度补充（2026-11-18，v0.87–v1.02）

> 不替换 `progress-snapshot-2026-11-17.md` 的 22 项定义；本节只校正 **R20/R21/runbook** 证据现状。

## R20

- **已具备**：schema8 双 HTTP 矩阵、settings 读写 CAS、混合/写读烟雾、`TestR20Schema8RunbookDefaultSmoke`、独立 workflow + validation 摘要 artifact（v1.02）。
- **仍不具备**：正式压测、p95/p99 签核 SLO、进行中 SSE 流级恢复、生产库双实例实测。

## R21

- **已具备**：探针表、试点表、签核记录模板、告警 handoff、实填指引。
- **仍不具备**：正式签字、监控规则生效、RPO/RTO 实测。

## 生产准备

- **仍缺**：runbook §3 生产实填、checklist 首条勾选（团队）；非代码可闭合项。

**进度百分比未据此上调**；见上一轮对话中的三维度评估（验证环境高 / 生产门低）。