# 第89轮：R20 模板复制 CAS 与 grant 授撤交叉

## 变更摘要

- **`TestR20Schema8DualHostTemplateCopyRejectsStaleRevision`**：模板草稿更新后双主机陈旧 `expected_template_revision` → 409。
- **`TestR20Schema8DualHostTemplateCopyConcurrentFresh`**：当前修订双主机并发复制，生成两个独立应用。
- **`TestR20Schema8DualHostGrantRevokePostRace`**：双主机 DELETE `grant-app` 与 POST 同义 grant 交叉，活跃授权 ≤1、审计增量受控。
- **R02**：`r02-mysql-api-evidence-v0.89.md`。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.89-template-grant-test.log`