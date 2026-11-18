# 第94轮：治理/消息读并发阶梯与 R21 SLO 草案

## 变更摘要

- **`TestR20Schema8TieredGovernanceUsageLoad`** / **`TestR20Schema8TieredMessageLoadLoad`**：32→64 workers 双主机阶梯，记录墙钟并设宽松上限（非正式 SLO）。
- **`mysql-schema8-r21-slo-draft.md`**：R21 正式 SLO 定标表（待团队签核）。
- runbook §3 增加「生产实填」行；§4 烟雾项挂接 v0.94；R02 v0.94。

## 测试

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8'
```

产物：`/opt/cursor/artifacts/v0.94-read-load-tier-test.log`