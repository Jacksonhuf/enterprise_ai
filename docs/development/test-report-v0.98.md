# 第98轮：settings 写矩阵与 public 写后混合烟雾

## 变更摘要

- **`TestR20Schema8DualHostSettingsProfileAndGeneralCAS`**：profile + general 双主机 PUT CAS（辅助函数复用）。
- **`TestR20Schema8MixedSmokeAfterPublicWrite`**：public 写入后混合读 bundle 校验公告一致。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.98-settings-smoke-test.log`