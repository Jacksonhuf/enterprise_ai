# 第124轮（v1.24）：容差第二轮收紧、exporter 评审形状、R18 manifest 对账

## 变更摘要

- `r18-tolerance-tightening-v1.24.json`（login/assets/services 窄屏）。
- `production-exporter-review-signed-shape-example.json`。
- `validate-r18-spot-manifest-sync.py`（**38** 抽页与 manifest/PNG 对账）。

## 测试

```bash
python scripts/validate-r18-spot-manifest-sync.py
python scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-v1.24.json
```