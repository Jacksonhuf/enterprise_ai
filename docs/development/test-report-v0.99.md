# 第99轮：混合烟雾写阶段与 R21 签核试点表

## 变更摘要

- **`TestR20Schema8MixedSmokeWriteThenReadOrchestration`**：public + profile 写阶段后双主机并发读（含 profile 校验）。
- **`mysql-schema8-r21-slo-pilot-signoff.md`**：签核前试点填表示例。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.99-write-read-smoke-test.log`