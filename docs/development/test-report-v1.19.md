# 第119轮（v1.19）：tier 联动 OpenMetrics、releases/usage 窄屏 R18

## 变更摘要

- `render-openmetrics-from-tier-summary.py` + `validate-openmetrics-tier-linkage.py`（CI）。
- Mock scrape 使用 **`COGNARA_OPENMETRICS_PATH=/tmp/openmetrics-from-tier.txt`**。
- R18：**35** 抽页（`build-releases-narrow`、`governance-usage-narrow`）。

## 测试

```bash
bash scripts/r09-tier-probes-and-summary.sh /tmp/t.log /tmp/tier.json
python scripts/render-openmetrics-from-tier-summary.py /tmp/tier.json /tmp/om.txt
python scripts/validate-openmetrics-tier-linkage.py /tmp/tier.json /tmp/om.txt
```