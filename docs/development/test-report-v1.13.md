# 第113轮（v1.13）：usage 96w、告警 6 规则、服务/资产窄屏 R18

## 变更摘要

- usage read **96w** 阶梯 + `r09-usage-tier-harness.sh`。
- dry-run 告警 **+tier96**（共 **6** 条），staging 工单 `rules_count=6`。
- R18：**23** 抽页（`services-narrow`、`assets-narrow`）；CI 窄屏单页容差见 `max-diff-ratio.json`。

## 测试

```bash
bash scripts/r09-usage-tier-harness.sh
python scripts/validate-r21-alerting-dry-run.py docs/development/fixtures/r21-alerting-dry-run.rules.json
```