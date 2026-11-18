# 第88轮：R20 发布检查/发布 CAS 与 grant 双主机撤权

## 变更摘要

- **`TestR20Schema8DualHostReleaseCheckRejectsStaleRevision`**：草稿更新后旧 `expected_draft_revision` 双主机 `release-checks` 均 409。
- **`TestR20Schema8DualHostReleasePublishCAS`**：同一 `check_id` 双主机 `POST .../releases`，1×200 + 1×409，发布与审计各 +1。
- **`TestR20Schema8DualHostConcurrentGrantRevoke`**：双主机并发 `DELETE` 同一 grant，撤权审计仅 +1。
- **R02**：`r02-mysql-api-evidence-v0.88.md`。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.88-release-grant-test.log`