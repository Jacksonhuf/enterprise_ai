# Release 门禁：R18 manifest 对账（v1.25）

在合入 `main` 或发版前，**F-G validation** 必须：

1. `validate-r18-spot-manifest-sync.py`：**38** 抽页 `scripts-smoke.mjs` ↔ `r18-baselines/manifest.json` ↔ PNG 一致。
2. `validate-r18-tolerance-tightening.py`（v1.23–v1.36 累积清单）与 `max-diff-ratio.json` 一致。
3. 下载 artifact **`r18-spot-screenshots`**、**`r18-pixel-tolerance-audit`**、**`r18-elevated-headroom-audit`**、**`r18-implicit-default-spot-catalog`**、**`r18-tolerance-tightening-rollup`**（分段归档，收口 v1.36）归档彩排。
4. `validate-r18-headroom-threshold-gate.py` 彩排告警（min **0.002**，`enforce_in_ci=false`）；违规见 headroom artifact `threshold_alerts`。

团队真 **CONFIRMED/APPROVED** 仍走 `engineering-pilot-bundle-*.json` 与 exporter 评审清单，**Agent 不代填真签**。