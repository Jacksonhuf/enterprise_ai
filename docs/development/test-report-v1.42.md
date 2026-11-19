# 第142轮（v1.42）：publish-review defer、窄屏子集、ENTERED/CONFIRMED 同桌

## 变更摘要

- `r18-publish-review-headroom-defer-v1.42.json`（v1.39 hold 0.031）。
- Headroom subset v1.42：`governance-grants-narrow`、`apps-narrow`（narrow_min 0.0005）。
- `ENTERED_CONFIRMED_BUNDLE_JOINT_SHAPE_ONLY`。

## 测试

```bash
python3 scripts/validate-r18-publish-review-headroom-defer.py docs/development/fixtures/r18-publish-review-headroom-defer-v1.42.json
python3 scripts/validate-production-gate-entered-confirmed-bundle-joint.py docs/development/fixtures/production-gate-entered-confirmed-bundle-joint-shape-example.json
python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-v1.42.json
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```