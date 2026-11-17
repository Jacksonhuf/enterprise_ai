# 第66轮：R06 schema4 快照 export/import

## 变更摘要

- `exportMySQLV4` / `parseMySQLV4`：封装 schema4 `state_root`、应用关系表、`cognara_fg_grants` 原文与审计。
- `ExportMySQLSnapshot` / `ImportMySQLSnapshot` 支持 `schema_version=4`；导入后 `OpenMySQL` 与 `readMySQLApplicationV4` 对账。
- 拒绝 root 内仍含 `grants` 的导出。

## 测试

```bash
cd backend/platform
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race \
  ./internal/platform/applications/ -run TestMySQLV4Snapshot -count=1
```

## 未在本轮

- R20 多实例 grant 并发专项
- R07 会话拆表