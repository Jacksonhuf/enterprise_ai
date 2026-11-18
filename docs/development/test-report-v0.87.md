# 第87轮：R20 grants 并发读 + 双主机异常 CAS + PAGE-F2 草稿 CAS

## 变更摘要

- **`TestR20Schema8ParallelGovernanceGrantsReads`**：双 HTTP 主机 32 路 `GET grants` 只读并发。
- **`TestR20Schema8DualHostExceptionStatusCAS`**：`exc-plan-1` 双主机 `POST .../status`，期望 1×200 + 1×409。
- **`TestR20Schema8DualHostApplicationDraftCAS`**：`PUT /api/v1/cognitive/apps/{id}` 双主机草稿 CAS（PAGE-F2）。
- **R02**：`r02-mysql-api-evidence-v0.87.md` 挂接 grants / exception status / app draft API。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.87-r20-grants-exc-draft-test.log`