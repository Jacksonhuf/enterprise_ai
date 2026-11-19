# 第140轮（v1.40）：chat-user 容差、headroom 子集门、签字名形状

## 变更摘要

- `chat-user` **0.025 → 0.026**；`forbidden-user-narrow` v1.28 defer。
- `validate-r18-headroom-threshold-subset-gate.py`（`publish-review` / `chat-user` 子集 enforce）。
- `CONFIRMED_SIGNATORY_NAMES_SHAPE_ONLY` bundle。

## 测试

```bash
python3 scripts/validate-r18-tolerance-tightening.py docs/development/fixtures/r18-tolerance-tightening-v1.40.json
python3 scripts/validate-r18-forbidden-user-narrow-headroom-defer.py docs/development/fixtures/r18-forbidden-user-narrow-headroom-defer-v1.40.json
python3 scripts/validate-engineering-pilot-bundle.py docs/development/fixtures/engineering-pilot-bundle-confirmed-signatory-names-shape-example.json
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```