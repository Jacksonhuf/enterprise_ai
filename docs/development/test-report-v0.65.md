# 第65轮：R06 schema4 授权关系 writer

## 变更摘要

- `TryMutateGrantsV4` / `mutateMySQLGrantsV4`：在 schema4 上通过 `cognara_fg_grants` 执行创建、去重与撤销，并与审计同事务提交。
- 治理 HTTP：`createGrant` / `revokeGrant` 在 schema4 下优先走路由，行为与聚合路径一致（去重无审计、撤销幂等）。
- 根 writer 仍禁止经 `state.body` 改 grants；变更仅允许关系 writer 路径。

## 测试

```bash
cd backend/platform
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race \
  ./internal/platform/applications/ -run MutateGrantsV4 -count=1
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race \
  ./internal/platform/api/ -run MySQLSchema4Grant -count=1
```

## 未在本轮

- schema4 快照导出/导入
- R07 会话拆表