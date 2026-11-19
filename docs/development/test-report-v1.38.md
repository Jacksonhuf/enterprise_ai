# 第138轮（v1.38）：收紧候选、headroom 开关、交叉表联席

## 变更摘要

- `report-r18-tolerance-tightening-candidates.py` + CI artifact（不改 `max-diff-ratio.json`）。
- Headroom 门 `r18-headroom-threshold-gate-v1.38.json`（`COGNARA_HEADROOM_ENFORCE_IN_CI` 团队开关）。
- `validate-production-gate-crossref-bundle-joint.py`（warehouse 联席）。

## 测试

```bash
python3 scripts/validate-r18-tolerance-tightening-candidates.py docs/development/fixtures/r18-tolerance-tightening-candidates-v1.38.json
python3 scripts/report-r18-tolerance-tightening-candidates.py /dev/null /tmp/candidates.json
python3 scripts/validate-production-gate-crossref-bundle-joint.py docs/development/fixtures/production-gate-crossref-bundle-joint-shape-example.json
python3 scripts/validate-r18-headroom-threshold-gate.py /tmp/fake-audit.json docs/development/fixtures/r18-headroom-threshold-gate-v1.38.json
python3 scripts/report-r18-tolerance-tightening-rollup.py /tmp/rollup.json
cd backend/platform && COGNARA_REQUIRE_MYSQL_TESTS=0 go test -p 1 -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -timeout 5m
```