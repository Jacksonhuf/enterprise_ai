# R18 缺口清单（v1.43）

> 38 个抽页在 CI 中的自动化状态与 **距 R18 关项** 的缺口。数据源：`frontend/r18-baselines/manifest.json`、`scripts-smoke.mjs`。

## 汇总

| 维度 | 已满足 | 仍缺 |
|------|--------|------|
| Chromium 烟测跑通 | 38/38 | — |
| PNG 基线 + manifest 同步 | 38/38 | — |
| 像素 diff 门禁 | 38/38 | — |
| 对照 Figma 像素签字 | 0/38 | 设计验收 |
| 键盘 / 焦点全路径 | 烟测未覆盖 | 全量 a11y |
| 空态 / 错态 / 加载态截图 | 夹具局部 | 逐页矩阵 |
| 真实身份导航壳 | demo 账户菜单 | SSO 后复测 |
| R18 任务关项签字 | — | 依赖 R02 字段级 + UAT |

## 按 slug 清单

| slug | 路径摘要 | 自动化 | 设计挂靠 | 仍缺 |
|------|----------|--------|----------|------|
| login-public | `/login` 宽 | 烟测+像素 | 补充-登录 | Figma、键盘 |
| login-public-narrow | 窄 | 同上 | 同上 | 同上 |
| build-apps-engineer | `/build/apps` 宽 | 同上 | F2-01/CAND-05 | 字段级 |
| build-apps-narrow | 窄 | 同上 | 同上 | 同上 |
| assets-engineer | `/assets` | 同上 | SETTINGS | R17 |
| assets-narrow | 窄 | 同上 | SETTINGS | 同上 |
| settings-tenant | tenant | 同上 | SETTINGS | Figma |
| settings-tenant-narrow | 窄 | 同上 | SETTINGS | 同上 |
| settings-models | models | 同上 | SETTINGS | 只读占位 |
| settings-models-narrow | 窄 | 同上 | SETTINGS | 同上 |
| governance-ops-settings | ops settings | 同上 | CAND-17 | 公告分发 |
| governance-ops-settings-narrow | 窄 | 同上 | CAND-17 | 同上 |
| governance-access | grants tab | 同上 | CAND-08–12 | 真实身份 |
| governance-audit | audit 页 | 同上 | CAND-13 | 投递回执 |
| governance-audit-narrow | 窄 | 同上 | CAND-13 | 同上 |
| forbidden-user | 拒绝 | 同上 | CAND-08 | SSO 后回归 |
| forbidden-user-narrow | 窄 | 同上 | 同上 | 同上 |
| apps-narrow | `/apps` | 同上 | PAGE-001 | 真链 |
| chat-user | 聊天入口 | 同上 | PAGE-001/002 | 真模型 |
| chat-user-narrow | 窄 | 同上 | 同上 | 同上 |
| editor-channel | 编辑 | 同上 | F2-02 | Figma |
| editor-channel-narrow | 窄 | 同上 | 同上 | 同上 |
| publish-review | 发布审阅 | 同上 | F2-02 | 容差紧 |
| publish-review-narrow | 窄 | 同上 | 同上 | headroom defer |
| app-detail-build | 详情 | 同上 | CAND-06 | Figma |
| app-detail-build-narrow | 窄 | 同上 | 同上 | 同上 |
| session-sidebar | 侧栏 | 同上 | PAGE-001 | 交互深度 |
| session-sidebar-narrow | 窄 | 同上 | 同上 | 同上 |
| build-releases | releases | 同上 | CAND-07 | — |
| build-releases-narrow | 窄 | 同上 | 同上 | 同上 |
| governance-usage | usage | 同上 | CAND-16 | 统计口径 |
| governance-usage-narrow | 窄 | 同上 | 同上 | 同上 |
| governance-ops-exceptions | exceptions | 同上 | CAND-15 | Figma |
| governance-ops-exceptions-narrow | 窄 | 同上 | 同上 | 同上 |
| governance-ops-audit | ops audit tab | 同上 | 补充 | 与 CAND-13 区分 |
| governance-ops-audit-narrow | 窄 | 同上 | 补充 | 同上 |
| services-narrow | services 仅窄 | 同上 | CAND-14 | **缺宽屏** |
| governance-grants-narrow | grants 窄 | 同上 | CAND-12 | 子集 headroom |

## 建议优先补洞（加速可见进度）

1. **CAND-14 services 宽屏 slug**（或产品确认仅窄屏验收）。
2. **CAND-09 / CAND-11** 独立 tab 抽页或书面接受「access 壳共用」。
3. **字段级核对表** 挂在 matrix JSON `gaps` 下钻链接（下一轮 v1.44）。

---

CI：`validate-r18-spot-manifest-sync.py` · 矩阵交叉：`validate-r02-page-api-acceptance-matrix.py`