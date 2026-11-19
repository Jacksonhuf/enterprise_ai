# 第123轮（v1.23）：窄屏容差收紧、生产 exporter 评审清单

## 变更摘要

- 治理窄屏 **usage/audit** 容差 **3% → 2.5%**（`r18-tolerance-tightening-v1.23.json`）。
- `mysql-schema8-production-exporter-readonly-review.md` 评审清单。
- CI：`validate-r18-tolerance-tightening.py`。

## 测试

全量见 v1.22 CI；本地跑窄屏像素门 + 上述校验脚本。