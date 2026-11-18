# R18 像素门阈值评审（v1.06）

## 当前门

| 项 | 值 |
|----|-----|
| 脚本 | `frontend/scripts-verify-r18-baseline.mjs` |
| 环境变量 | `COGNARA_R18_MAX_DIFF_RATIO`（默认 **0.02** = 2%） |
| 按面覆盖 | `frontend/r18-baselines/max-diff-ratio.json`（v1.08：`publish-review` 3%） |
| pixelmatch | `threshold: 0.2`, `includeAA: false` |

## 何时调高/调低

| 现象 | 建议 |
|------|------|
| CI 仅 1–2 页超 2%、差异为抗锯齿/字体 | 暂保持 2%；在 PR 中 **刷新基线** 并附 UI 变更说明 |
| 多页持续 >3% 无预期 UI 改 | 先查 TDesign/主题回归，勿盲目放宽 |
| 故意大改版 | 重跑 `COGNARA_R18_CAPTURE=./r18-baselines npm run test:cognara-browser` 并提交 PNG |

## 抽页规模（v1.06）

10 面：login、build/assets（工程师）、原 7 面 + 窄屏 apps。

全量 R18 仍需设计与人工签字。