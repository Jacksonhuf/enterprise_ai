# 第73轮：R08 schema6 快照

## 变更摘要

- `exportMySQLV6` / `parseMySQLV6` / `stateFromV6Snapshot`：根域不含 settings 聚合字段，关系行单独保真。
- 公共 `ExportMySQLSnapshot` / `ImportMySQLSnapshot` 支持 schema6（应用/grant/session/settings 表 + 审计）。
- `applyMySQLSettingsSnapshot`：空库导入 settings 行。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'V6Snapshot'
COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race ./internal/platform/applications/
```

## 未在本轮

- 专用 bulk settings writer
- exceptions / usage 关系域（R08 后续切片）