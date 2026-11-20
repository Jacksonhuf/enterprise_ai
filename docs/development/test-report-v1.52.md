# 测试报告 v1.52（演练表 C8 + gap 登记 + Figma 18）

## 范围

- `schema8-runbook-first-fill-rehearsal-table-v1.52.json`（C8、log_align）
- matrix **V150** + `r02-page-api-open-gap-registry-v1.52.json`
- Figma **V152**，≥18 matched

## 本地

```bash
python3 scripts/validate-runbook-first-fill-rehearsal-table.py docs/development/fixtures/schema8-runbook-first-fill-rehearsal-table-v1.52.json
python3 scripts/validate-runbook-log-rehearsal-alignment.py docs/development/fixtures/runbook-log-rehearsal-alignment-v1.52.json
python3 scripts/validate-r02-page-api-open-gap-registry.py docs/development/fixtures/r02-page-api-open-gap-registry-v1.52.json
```

## 结论

工程彩排；开放 gap 登记不代表关项。