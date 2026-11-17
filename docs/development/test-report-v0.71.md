# 第71轮：R08 settings plan 与 DDL

## 变更摘要

- `migrations/005_settings.sql`：`cognara_fg_settings` 目标表与 CHECK。
- `splitSettingsLayout` / `joinSettingsLayout` / `verifySettingsLayout`：要求 schema5 根域；`settings` 与 `setting_revisions` 对象成对。
- 只读 `PlanMySQLSettings`（要求 schema5 + 关系表齐备；目标 schema6；拒绝保留表已存在）。
- `cognara-migrate-settings --check`；内部 `rehearseMySQLSettingsMigration` 复制后回滚。
- CI 纳入 `cognara-migrate-settings` 构建与测试。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'Settings|settings' \
  ./cmd/cognara-migrate-settings/
```

## 未在本轮

- `ApplyMySQLSettings` / `--apply` 与 schema6 运行时
- exceptions / usage 关系域（R08 后续切片）