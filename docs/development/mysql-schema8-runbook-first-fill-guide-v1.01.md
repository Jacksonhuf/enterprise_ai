# 生产 runbook 首条实填与签核验收指引（v1.01）

面向运维/平台工程：**首条生产 apply** 完成后的登记与证据挂载。

## 1. 自动化证据（CI 可选）

| 来源 | 用途 |
|------|------|
| **F-G runbook default smoke**（`fg-runbook-default-smoke.yml`） | `workflow_dispatch` 或相关路径 push main 后单独跑默认门；Run URL 可贴入 checklist §4 |
| **F-G validation** | 全量门禁；PR 合并前必备 |

## 2. 人工勾选

使用 [mysql-schema8-runbook-production-fill-checklist.md](mysql-schema8-runbook-production-fill-checklist.md) 逐项勾选；§3 与 `mysql-schema8-production-apply-runbook.md` §3「生产实填」同步。

## 3. R21 签核

1. 核对 `mysql-schema8-r21-slo-pilot-signoff.md` 试点墙钟与生产探针（若已跑）。
2. 在 `mysql-schema8-r21-slo-signoff-record.md` 签字并填写监控规则版本。
3. 在 runbook §3「SLO 签核」行登记工单号。

**本指引不替代签字**；未签字前不得对外宣称 SLO 已生效。