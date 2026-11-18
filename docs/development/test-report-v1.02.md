# 第102轮（v1.02）：validation runbook 摘要 artifact

## 变更摘要

- **`scripts/collect-runbook-smoke-summary.py`**：从 go test 日志提取默认门状态与写/读墙钟。
- **`fg-validation.yml`**：平台测试 tee 日志 → 摘要 JSON → artifact `runbook-smoke-summary-validation`。
- **`fg-runbook-default-smoke.yml`**：上传 `runbook-default-smoke`（log + summary）。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke -v 2>&1 | tee /tmp/log
python ../scripts/collect-runbook-smoke-summary.py /tmp/log /tmp/summary.json
```

产物示例：`/opt/cursor/artifacts/runbook-smoke-summary.json`