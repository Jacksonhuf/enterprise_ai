# 第128轮（v1.28）：第六轮容差、G4 SHA256 附录

## 变更摘要

- `r18-tolerance-tightening-v1.28.json`（`app-detail-build-narrow`、`forbidden-user-narrow`）。
- 台账 ENTERED 形状含 `runbook_smoke_summary_sha256`；exporter **附录 A** + g4 appendix JSON。

## 测试

```bash
for v in v1.23 v1.24 v1.25 v1.26 v1.27 v1.28; do
  python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-$v.json
done
python3 scripts/validate-r18-spot-manifest-sync.py
python3 scripts/validate-production-ledger.py docs/development/fixtures/schema8-production-ledger-{pending,entered-shape}-example.json
python3 scripts/validate-production-exporter-review.py docs/development/fixtures/production-exporter-review-g4-appendix-shape-example.json
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```