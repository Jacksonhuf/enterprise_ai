# 第76轮：R08 schema7 快照

## 变更摘要

- `exportMySQLV7` / `parseMySQLV7` / `stateFromV7Snapshot`：根域不含 exceptions 聚合字段，关系行单独保真。
- 公共 `ExportMySQLSnapshot` / `ImportMySQLSnapshot` 支持 schema7（应用/grant/session/settings/exceptions 表 + 审计）。
- `applyMySQLExceptionSnapshot`：空库导入 exceptions 行；导入后 `readMySQLApplicationV7` 对账。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'V7Snapshot'
COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race ./internal/platform/applications/
```

产物：`/opt/cursor/artifacts/exceptions-snapshot-test.log`（全包 `./internal/platform/applications/` 通过）。

## 未在本轮

- usage 关系域（计划 schema8）
- 专用 bulk exceptions writer（仍拒绝根域整段写）
- R20 多实例并发与 R07 分页/脱敏专项