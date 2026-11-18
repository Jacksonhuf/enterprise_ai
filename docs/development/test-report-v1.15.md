# 第115轮（v1.15）：ENTERED 台账形状、recording rule 样例、设置窄屏 R18

## 变更摘要

- `schema8-production-ledger-entered-shape-example.json`。
- `r21-recording-rules-draft.yaml` + `validate-r21-recording-rules-draft.py`。
- R18：**27** 抽页（`settings-tenant-narrow`、`settings-models-narrow`）。

## 测试

```bash
pip install pyyaml==6.0.2
python scripts/validate-r21-recording-rules-draft.py docs/development/fixtures/r21-recording-rules-draft.yaml
```