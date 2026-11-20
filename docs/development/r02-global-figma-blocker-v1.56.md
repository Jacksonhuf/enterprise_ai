# R02 全局 Figma 阻塞说明（v1.56）

## 背景

- **字段级**：Figma checklist 已达 `REHEARSAL_ALL_FIELDS_MATCHED`（22/22 engineering matched，彩排证据见 `r02-figma-field-checklist-v1.44.json`）。
- **全局级**：验收矩阵 `r18_global_gaps` 仍含 **`FIGMA_SIGNOFF_ALL`**，表示产品设计/视觉负责人尚未对全平台范围签字关项。

二者不矛盾：engineering matched 只说明自动化清单与工程证据对齐，**非产品签字**。

## 仍开放的典型全局 gap（与 matrix 一致）

| Gap ID | 含义 |
|--------|------|
| `FIGMA_SIGNOFF_ALL` | 全范围 Figma/产品签收 |
| `KEYBOARD_A11Y_FULL` | 键盘与可访问性全态 |
| `EMPTY_ERROR_LOADING_ALL_STATES` | 空/错/加载等全状态截图闭环 |
| `REAL_IDENTITY_NAV` | 真实身份与导航 |
| `PRODUCTION_UAT_SIGNOFF` | 联合 UAT / 发布签字 |

closeout（`r02-page-api-gap-closeout-v1.49.json`）仅覆盖有限工程收口项，**不关闭**上表全局阻塞。

## 联合门（v1.56）

`validate-r02-global-gap-dashboard-matrix-joint.py` 要求同时满足：

1. 全局 gap 可视化与 open gap registry 一致；
2. matrix↔Figma 双向链路仍指向 V155 满编彩排；
3. 本说明文档存在且显式区分「engineering matched」与「FIGMA_SIGNOFF_ALL」。

## 生产台账

生产 §3 log 仍以 SRE 真签为准；`schema8-production-log-rehearsal-filled-v1.56.json` 为 **REHEARSAL_ONLY** 实填彩排，勿写入生产库。