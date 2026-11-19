# 测试报告 v1.50（matrix↔Figma 双向 + 首条真填演练表）

## 范围

- `validate-r02-matrix-figma-bidirectional.py`（matrix V149 + figma V150，≥14 matched）
- `schema8-runbook-first-fill-rehearsal-table-v1.50.json` 链式校验
- Figma **15** matched（V150）

## 本地

```bash
python3 scripts/validate-r02-matrix-figma-bidirectional.py docs/development/fixtures/r02-matrix-figma-bidirectional-link-v1.50.json
python3 scripts/validate-runbook-first-fill-rehearsal-table.py docs/development/fixtures/schema8-runbook-first-fill-rehearsal-table-v1.50.json
```

## 结论

工程彩排；**非**生产 §3 真填。