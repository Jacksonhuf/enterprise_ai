# 第135轮（v1.35）：slug 编目、headroom artifact、交叉表模板

## 变更摘要

- `r18-tolerance-tightening-v1.35.json`（`apps-narrow`、`governance-ops-exceptions-narrow`）。
- 像素门写出 `COGNARA_R18_PIXEL_MEASURED`；`report-r18-elevated-headroom-audit.py`。
- `production-gate-ticket-crossref-pending-template.json`。

## 测试

```bash
for v in v1.23 v1.24 v1.25 v1.26 v1.27 v1.28 v1.29 v1.30 v1.31 v1.33 v1.34 v1.35; do
  python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-$v.json
done
python3 scripts/validate-production-gate-ticket-crossref.py docs/development/fixtures/production-gate-ticket-crossref-pending-template.json
python3 scripts/report-r18-elevated-headroom-audit.py /dev/null /tmp/headroom.json
python3 scripts/validate-r18-spot-manifest-sync.py
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```