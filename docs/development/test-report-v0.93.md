# 第93轮：continue-stream 并发重放与 SLO 探针基线

## 变更摘要

- **`TestR20Schema8ParallelContinueStreamReplay`**：32 路双主机 `continue-stream` 重放，业务态不变。
- **`mysql-schema8-r20-slo-probe-baseline.md`**：轻量探针登记（非正式 SLO）。
- runbook §4 烟雾项与 R02 v0.93。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.93-sse-probe-test.log`