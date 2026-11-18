# 第112轮（v1.12）：message load 96w 阶梯、治理窄屏 R18、SLO SIGNED 形状

## 变更摘要

- `TestR20Schema8TieredMessageLoadLoad`：**32 / 64 / 96** workers。
- `r09-message-load-tier-harness.sh`（CI）。
- R18：**21** 抽页（治理 grants/异常 **390px**）。
- `r21-slo-signoff-record-signed-shape-example.json`。

## 测试

```bash
bash scripts/r09-message-load-tier-harness.sh
# browser + 默认门：同前几轮
```