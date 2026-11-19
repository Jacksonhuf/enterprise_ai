# 第125轮（v1.25）：第三轮容差、APPROVED 形状、release 门禁文档

## 变更摘要

- `r18-tolerance-tightening-v1.25.json`（build-releases / governance-usage 窄屏）。
- Exporter **`APPROVED_SHAPE_ONLY`**、彩排 bundle **`APPROVED_SHAPE_ONLY`** 样例。
- `mysql-schema8-r20-release-gate-r18-manifest.md`。

## 测试

```bash
python scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-v1.25.json
python scripts/validate-r18-spot-manifest-sync.py
```