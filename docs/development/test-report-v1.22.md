# Test report v1.22

## Scope

- `validate-openmetrics-ci-snapshot-full.py`: 6 指标集合 + tier/runbook/mixed 联动（CI 真值路径）。
- `report-r18-pixel-tolerance-audit.py`: 高于 default 2% 的 R18 容差盘点 JSON artifact。
- Engineering pilot bundle：`openmetrics_linked_metric_count: 6`。

## Local checks

```bash
python scripts/validate-engineering-pilot-bundle.py docs/development/fixtures/engineering-pilot-bundle-pending-example.json
python scripts/validate-engineering-pilot-bundle.py docs/development/fixtures/engineering-pilot-bundle-confirmed-shape-example.json
python scripts/report-r18-pixel-tolerance-audit.py /tmp/r18-pt-audit.json
```

全量 snapshot 联动验收在 CI `fg-validation` 中与 tier/runbook/mixed 探针同一 job 内执行。

## Not in scope

- 团队真 CONFIRMED / ENTERED 签字名（仍为 PLACEHOLDER 形状样例）。
- 生产 scrape 与正式台账入账。