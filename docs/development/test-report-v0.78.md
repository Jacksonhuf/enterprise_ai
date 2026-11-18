# 第78轮：R08 usage apply 与 schema8 读

## 变更摘要

- `ApplyMySQLUsage`、`cognara-migrate-usage --apply --expect-sha256`：schema7→8，usage 迁入 `cognara_fg_usage`。
- `readMySQLApplicationV8` 接入 `OpenMySQL`、根域 guard（禁止经 body 改 usage；追加走 `applyMySQLUsageAppendDelta`）。
- schema8 下治理 `usage` 列表走 `queryMySQLUsageRelational`。
- schema8 下 exceptions 状态更新仍走关系表（`readMySQLApplicationRelational` 对账）。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'ApplyMySQLUsage|Usage' \
  ./cmd/cognara-migrate-usage/
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race ./internal/platform/applications/
```

产物：`/opt/cursor/artifacts/usage-apply-test.log`

## 未在本轮

- schema8 快照 export/import
- R20 多实例并发与 R07 分页/脱敏专项