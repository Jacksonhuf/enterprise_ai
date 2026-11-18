# 第80轮：R20 schema8 双连接并发（首轮）

## 变更摘要

在 schema8 全迁移夹具上新增 R20 首轮专项（共享 MySQL DSN、双 `OpenMySQL` 连接）：

- `TestR20Schema8DualStoreDraftCAS`：并发草稿 CAS（一成功、一 `ErrRelationalDraftConflict`）。
- `TestR20Schema8DualStoreConcurrentUsageAppends`：独立 `event_id` 用量追加不丢事件。
- `TestR20Schema8DualStoreExceptionStatusSerialize`：并发异常状态更新不重复行、终态合法。

## 测试

```bash
cd backend/platform
COGNARA_TEST_MYSQL_DSN='root:validation-only-mysql-password@tcp(127.0.0.1:3306)/cognara_validation' \
COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race -count=1 \
  ./internal/platform/applications/ -run 'R20Schema8'
COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1 go test -p 1 -race ./internal/platform/applications/
```

产物：`/opt/cursor/artifacts/r20-schema8-concurrency-test.log`

## 未在本轮

- 双 `cognara-dev` HTTP 主机/SSE 中断与压力测量
- 正式 SLO 对照与运维登记
- grant/settings 全量并发矩阵