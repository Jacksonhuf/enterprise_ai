# 第72轮：R08 settings apply 与 schema6 读

## 变更摘要

- `ApplyMySQLSettings`：`stageMySQLSettingsRoot`、schema_version 5→6、提交后 `readMySQLApplicationV6` 对账。
- `cognara-migrate-settings --apply --expect-sha256`（镜像 sessions/grants CLI）。
- `readMySQLApplicationV6` 接入 `OpenMySQL`、`readMySQLApplicationRelational`、grant/session writer 与根域 writer（禁止经 body 改 settings）。
- schema6 下 `UpdateMySQLSettingsContext` / `readMySQLSettingsProjection` 走 `cognara_fg_settings` 关系行。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'Settings|settings' \
  ./cmd/cognara-migrate-settings/
COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race ./internal/platform/applications/
```

## 未在本轮

- schema6 专用 settings writer（除 JSON 路径外的批量变更仍经根域拒绝）
- schema6 快照 export/import
- exceptions / usage 关系域（R08 后续切片）