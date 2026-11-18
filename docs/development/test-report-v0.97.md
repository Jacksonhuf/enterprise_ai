# 第97轮：settings/public 双主机写 CAS 与进行中 turn 断连语义

## 变更摘要

- **`TestR20Schema8DualHostSettingsPublicCAS`**：企业 public 配置双主机 PUT CAS。
- **`TestR20Schema8DualHostContinueStreamInProgressTurn`** / **`TestR20Schema8InProgressTurnDisconnectThenCompletedReplay`**：进行中消息不可重放；断连后可完成并重放。
- **`mysql-schema8-inprogress-turn-semantics.md`**：R14 探针边界说明。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.97-settings-inprogress-test.log`