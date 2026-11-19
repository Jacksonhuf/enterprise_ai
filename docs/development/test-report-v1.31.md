# 第131轮（v1.31）：第九轮容差、bundle↔ledger 工单对账

## 变更摘要

- `r18-tolerance-tightening-v1.31.json`（governance-ops-settings/audit narrow）。
- `apply_change_ticket_ref` on bundle + ledger；`validate-pilot-ledger-bundle-ticket-sync.py`。

## 测试

```bash
for v in v1.23 v1.24 v1.25 v1.26 v1.27 v1.28 v1.29 v1.30 v1.31; do
  python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-$v.json
done
python3 scripts/validate-pilot-ledger-bundle-ticket-sync.py \
  docs/development/fixtures/engineering-pilot-bundle-confirmed-shape-example.json \
  docs/development/fixtures/schema8-production-ledger-entered-shape-example.json
python3 scripts/validate-r18-spot-manifest-sync.py
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```