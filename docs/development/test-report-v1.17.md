# 第117轮（v1.17）：tier 入账打包、编辑器/发布窄屏 R18

## 变更摘要

- `validate-tier-probe-summary.py`（CI 真值 + 形状样例）。
- `engineering-pilot-bundle-pending-example.json` 关联 runbook / tier / R18 artifacts。
- R18：**31** 抽页（`editor-channel-narrow`、`publish-review-narrow`）。

## 测试

```bash
python scripts/validate-engineering-pilot-bundle.py docs/development/fixtures/engineering-pilot-bundle-pending-example.json
```