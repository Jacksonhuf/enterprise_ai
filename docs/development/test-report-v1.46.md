# 测试报告 v1.46（三表同窗 + 真签拒 SHAPE + Figma 3 matched）

## 范围

- `production-gate-three-table-window-v1.46.json` + crossref/staging CLOSED v146
- `validate-production-gate-shape-rejection-gate.py`（负例必须失败）
- Figma checklist **3** matched（V146）

## 本地

```bash
python3 scripts/validate-production-gate-three-table-window.py docs/development/fixtures/production-gate-three-table-window-v1.46.json
python3 scripts/validate-production-gate-shape-rejection-gate.py docs/development/fixtures/production-gate-shape-rejection-gate-v1.46.json
python3 scripts/validate-r02-figma-field-checklist.py docs/development/fixtures/r02-figma-field-checklist-v1.44.json
```

## 结论

工程彩排：bundle/ledger/staging/crossref **同窗**；禁止 PLACEHOLDER/SHAPE 冒充生产签字 claim。**非**团队真签。