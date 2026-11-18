# 第79轮：R08 schema8 快照

## 变更摘要

- `exportMySQLV8` / `parseMySQLV8` / `stateFromV8Snapshot`：根域不含 usage 聚合字段，关系行单独保真。
- 公共 `ExportMySQLSnapshot` / `ImportMySQLSnapshot` 支持 schema8（应用/grant/session/settings/exceptions/usage 表 + 审计）。
- `applyMySQLUsageSnapshot`：空库导入 usage 行；导入后 `readMySQLApplicationV8` 对账。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'V8Snapshot'
COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race ./internal/platform/applications/
```

产物：`/opt/cursor/artifacts/usage-snapshot-test.log`

## 里程碑

R08 settings / exceptions / usage 在代码侧已形成 **迁移 → apply → 关系读写 → 无损快照** 全闭环。

## 未在本轮

- R20 多实例 MySQL 并发
- R07 分页/脱敏生产验收
- receipts / outbox 等后续关系域