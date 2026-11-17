# 第64轮：R06 授权显式 apply 与 schema4 运行时

## 变更摘要

- `ApplyMySQLGrants`：plan SHA 校验后单事务 `stageMySQLGrantRoot` + `schema_version=4`
- `readMySQLApplicationV4` / `readMySQLApplicationRelational`：`sql_store` 与 `OpenMySQL` 在 schema4 下重组态
- `updateMySQLV3RootTx`：schema4 禁止经根 writer 改 grants（`errRelationalGrantDomain`）
- `cognara-migrate-grants --apply --expect-sha256`
- 应用关系写路径（create/draft/check/publish）与观测路径在 schema4 复用 v3 关系表

## 测试

本地（`COGNARA_REQUIRE_MYSQL_TESTS=1`，`cognara_validation`）：

```bash
cd backend/platform
go test -p 1 -race ./cmd/cognara-migrate-grants/ \
  ./internal/platform/applications/ -run 'Grant|ApplyMySQLGrants|GrantPlanner|GrantMigration' -count=1
```

覆盖：grant plan/apply、schema4 `OpenMySQL`、根域 grants 剥离、重复 apply 拒绝、CLI 门禁。

## 未在本轮范围

- grants 关系 writer / HTTP 专项路径
- schema4 快照导出/导入（export 仍限 v1–v3）
- R07 会话拆表