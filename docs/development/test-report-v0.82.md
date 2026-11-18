# 第82轮：R20 schema8 grant/settings 双 HTTP 矩阵

## 变更摘要

- **R20 `TestR20Schema8DualHostConcurrentGrants`**：双 httptest 主机复用 Postgres 专项逻辑——并发创建 grant 去重、幂等撤销与再授权审计。
- **R20 `TestR20Schema8DualHostSettingsProfileCAS`**：schema8 关系化 settings 上双主机 profile CAS（1×200、1×409）。
- **修复**：`UpdateMySQLSettingsContext` / `readMySQLSettingsProjection` 在 schema 7–8 走关系化 upsert/读路径（此前仅 v6）。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/
```

产物：`/opt/cursor/artifacts/r20-schema8-grant-settings-http-test.log`

## 未在本轮

- R20 压力/SSE/SLO
- 生产 apply 运维登记