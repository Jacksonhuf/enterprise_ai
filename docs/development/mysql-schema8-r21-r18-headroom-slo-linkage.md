# R18 像素 headroom 与 R21 SLO 联动（v1.37）

工程彩排将 CI 像素门 **headroom**（`max_diff_ratio − measured_ratio`）与 SLO 草案对照，**不构成对外 SLO**。

## 阈值与告警

| 项 | 值 |
|----|-----|
| 彩排最小 headroom | **0.002**（`r18-headroom-threshold-gate-v1.36.json`） |
| CI 强制 | `enforce_in_ci=false`（违规写入 `r18-elevated-headroom-audit` 的 `threshold_alerts`） |
| 团队开关（v1.38） | 环境变量 `COGNARA_HEADROOM_ENFORCE_IN_CI=1` + `r18-headroom-threshold-gate-v1.38.json` |
| 专项 workflow（v1.39） | **Actions → F-G headroom enforce rehearsal**（`fg-headroom-enforce-rehearsal.yml`） |
| 联动 fixture | `r18-headroom-slo-linkage-v1.37.json` |

## 与 SLO 草案关系

1. **R18 像素门** — 见 `mysql-schema8-r20-release-gate-r18-manifest.md`；近顶 spot 提示下一轮容差编目或基线刷新。
2. **R21 SLO 草案** — 见 `mysql-schema8-r21-slo-draft.md` §2；团队签核前将 headroom 盘点附在变更单证据包。
3. **正式签核** — 仍走 `mysql-schema8-r21-slo-signoff-record.md`；Agent 不代填。

## runbook 勾选项

`mysql-schema8-production-apply-runbook.md` G2：容差 rollup 收口 **v1.37** 且 headroom artifact 无未处置的 `threshold_alerts`（团队判定，非 Agent 自动 sign-off）。