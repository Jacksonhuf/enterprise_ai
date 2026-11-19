# 第121轮（v1.21）：mixed-read→OpenMetrics、窄屏覆盖率 artifact

## 变更摘要

- mixed-read probe harness + 写入 **`openmetrics-ci-snapshot`**。
- 彩排 bundle 增加 `openmetrics-ci-snapshot` artifact 名。
- CI **`r18-narrow-coverage`** artifact。

## 测试

```bash
bash scripts/r09-mixed-read-probe-harness.sh /tmp/m.log /tmp/m.json
python scripts/render-openmetrics-from-mixed-read-summary.py /tmp/m.json /tmp/openmetrics-ci-snapshot.txt
```