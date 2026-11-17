# 第70轮：R07 schema5 快照 export/import

## 变更摘要

- `exportMySQLV5` / `parseMySQLV5`：封装 schema5 `state_root`、应用关系表、grants、sessions/messages 原文与审计。
- `ExportMySQLSnapshot` / `ImportMySQLSnapshot` 支持 `schema_version=5`；导入后 `OpenMySQL` 与 `readMySQLApplicationV5` 对账。
- 拒绝 root 内仍含 `sessions` 的导出；`stateFromV3Snapshot` 允许空 sessions map 以支持 v5 根域。

## 测试

```bash
cd backend/platform
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race \
  ./internal/platform/applications/ -run TestMySQLV5Snapshot -count=1
```

## 未在本轮

- 会话列表分页与撤权脱敏
- R07 整项生产切换与 R20 多实例并发