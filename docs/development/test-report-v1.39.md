# 第139轮（v1.39）：services-narrow 容差、headroom 专项 workflow、CONFIRMED 联席

## 变更摘要

- `r18-tolerance-tightening-v1.39.json`：`publish-review` **0.03 → 0.031**；`services-narrow` 因 v1.34 hold 推迟。
- `.github/workflows/fg-headroom-enforce-rehearsal.yml`（`workflow_dispatch` + `COGNARA_HEADROOM_ENFORCE_IN_CI`）。
- `CROSSREF_BUNDLE_CONFIRMED_JOINT_SHAPE_ONLY` + confirmed warehouse bundle fixture。

## 测试

```bash
python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-v1.39.json
python3 scripts/validate-production-gate-crossref-bundle-joint.py docs/development/fixtures/production-gate-crossref-bundle-confirmed-joint-shape-example.json
python3 scripts/validate-engineering-pilot-bundle.py docs/development/fixtures/engineering-pilot-bundle-confirmed-warehouse-ticket-shape-example.json
# headroom enforce rehearsal (same steps as fg-headroom-enforce-rehearsal.yml)
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```