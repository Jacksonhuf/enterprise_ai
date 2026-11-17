# 第75轮：R08 exceptions apply 与 schema7 读

## 变更摘要

- `ApplyMySQLExceptions`、`cognara-migrate-exceptions --apply --expect-sha256`：schema6→7，exceptions 迁入 `cognara_fg_exceptions`。
- `readMySQLApplicationV7` 接入 `OpenMySQL`、grant/session writer 与根域 guard（禁止经 body 改 exceptions）。
- schema7 下异常状态更新、治理列表查询与失败观测追加走关系表。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'ApplyMySQLExceptions|Exception' \
  ./cmd/cognara-migrate-exceptions/
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race ./internal/platform/applications/
```

## 未在本轮

- schema7 快照 export/import
- usage 关系域