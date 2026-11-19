# R02：页面 / API 验收矩阵（v1.43）

> **口径**：17 个 F/G 设计条目逐条挂路由、源码视图、R18 抽页 slug、主要 API 与缺口码；**不**代表 R02 整项关闭。机器可读副本：`fixtures/r02-page-api-acceptance-matrix-v1.43.json`；CI 校验：`validate-r02-page-api-acceptance-matrix.py`。

## 进度一览（可见指标，非产品百分比）

| 指标 | 数值 | 说明 |
|------|------|------|
| 设计条目 | **17/17** 已登记路由 + 视图 + slug | 含 SETTINGS 挂靠与补充映射 |
| R18 slug 覆盖 | **38/38** 已挂靠设计或补充项 | 与 `manifest.json` 一致 |
| 像素回归 CI | **38/38** | `max-diff-ratio.json` + 像素门 |
| Mock 演示身份链 | **17/17** | `link_mode` 均为 mock/demo |
| 真实身份 / 真实 A–E | **0/17** 未验收 | R09–R12 阻塞 |
| Figma 逐字段签字 | **0/17** | 缺口码 `FIGMA_SIGNOFF` |
| 专眼红点（无专属抽页） | **3** | 见下表「无专属 R18」 |

缺口码：`FIGMA_SIGNOFF` · `FIELD_BY_FIELD` · `REAL_IDENTITY` · `REAL_MODEL_AE` · `KEYBOARD_A11Y` · `NO_R18_DEDICATED_SPOT` · `TAB_PIXEL_GAP` · `NO_WIDE_R18_SPOT` · 等（见 JSON）。

## 17 项设计条目矩阵

| ID | 路由（代表） | 视图 | R18 slug（摘要） | 联调 | 主要缺口 |
|----|-------------|------|------------------|------|----------|
| PAGE-001 | `/apps` | Applications.vue | apps/chat/session 窄+宽 | Mock | 真模型/A–E、Figma |
| PAGE-002 | `/apps/:id/chat/:sessionId` | Chat.vue | chat-user ×2 | Mock | 远端取消、Figma |
| PAGE-F2-01 | `/build/apps` | Applications.vue | build-apps ×2 | Mock | 字段核对 |
| PAGE-F2-02 | `/build/apps/:id/edit` | Editor.vue | editor/publish-review ×4 | Mock | 真引用、键盘 |
| CAND-05 | `/build/apps` | Applications.vue | 同 F2-01 | Mock | 字段核对 |
| CAND-06 | `/build/apps/:id` | ApplicationDetail.vue | app-detail ×2 | Mock | Figma |
| CAND-07 | `?tab=releases` | ApplicationDetail.vue | build-releases ×2 | Mock | 非线上回滚 |
| CAND-08 | access `tab=users` | Governance.vue | access/forbidden/grants 窄 | Demo 列表 | **真实身份** |
| CAND-09 | `tab=groups` | Governance.vue | 借 access 壳 | Demo | **无专属 R18** |
| CAND-10 | `tab=clients` | Governance.vue | grants 窄 | Demo | Tab 像素 |
| CAND-11 | `tab=roles` | Governance.vue | 借 access 壳 | Demo | **无专属 R18** |
| CAND-12 | `tab=grants` | Governance.vue | grants 窄 | Mock CAS | 真实主体 |
| CAND-13 | `/governance/audit` | Governance.vue | audit ×2 | Mock | 跨服务投递 |
| CAND-14 | `tab=services` | Governance.vue | **仅** services-narrow | Mock | **无宽屏抽页** |
| CAND-15 | `tab=exceptions` | Governance.vue | exceptions ×2 | Mock | Figma |
| CAND-16 | `tab=usage` | Governance.vue | usage ×2 | Mock | 非账单 |
| CAND-17 | settings tab + `/settings/:section` | Governance.vue, Settings.vue | ops-settings 窄+宽 | Mock | 无全员公告 |

**挂靠项（不计入 17）**

| ID | 路由 | R18 slug | 缺口 |
|----|------|----------|------|
| SETTINGS-SECTIONS | `/settings/*`, `/assets`, `/services` | models, assets, tenant | R17 逐项启用 |
| 补充 | `/login`, ops `tab=audit` | login ×2, ops-audit ×2 | 键盘/Figma |

## API 证据索引（延续 v0.85+）

| 域 | 代表 API | 证据 |
|----|----------|------|
| 聊天/会话 | `continue-stream`, `messages/load`, session CAS | v0.85、schema8 R20 双机 |
| 授权 | grants POST/DELETE | v0.85、Chromium 隔离 |
| 用量 | `governance/usage` | v0.81 关系分页 |
| 设置/公告 | settings PUT、public | R01 PR40、Chromium |
| 全量契约 | — | `cognara-contract.jsonl`（fg-validation） |

历史增量索引链：`r02-mysql-api-evidence-v0.85.md` → `v1.42.md`；本 matrix **不重写** 逐 API 测试表。

## 与 R18 的关系

- R18 **全页**验收仍未关闭；38 抽页仅缩小 **R03/R02 视觉风险**。
- 下一批 R02 工作（建议）：为 CAND-09/11/14 增 **专署 slug** 或登记「接受壳层共用」产品签字；启动 **字段级** 核对表（Figma ↔ 控件）。

---

关联：[R18 缺口清单 v1.43](r18-gap-checklist-v1.43.md) · [进度补充 2026-11-19](progress-snapshot-2026-11-19-r02-r18.md) · [test-report-v1.43.md](test-report-v1.43.md)