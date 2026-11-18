# schema8 发布后默认烟雾命令（v1.00+）

与 `mysql-schema8-production-apply-runbook.md` §4 对齐。双 Go 实例共享 DSN 的 apply 后，**优先**跑默认门，再按需跑完整 `R20Schema8` 矩阵。

## 默认门（必选）

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ \
  -run TestR20Schema8RunbookDefaultSmoke -timeout 10m
```

覆盖：public + profile 写阶段、双主机并发读（治理/配置/消息）；见 `TestR20Schema8MixedSmokeWriteThenReadOrchestration`。

## 扩展矩阵（建议同日执行）

```bash
go test -p 1 -race -count=1 ./internal/platform/applications/
go test -p 1 -race -count=1 ./internal/platform/api/ -run 'R20Schema8|R07Schema8' -timeout 30m
```

证据：`test-report-v0.99.md`、各轮 `test-report-v0.8x–0.99`。

## CI 可选 job（v1.01+）

仓库 **Actions → F-G runbook default smoke**（`workflow_dispatch` 或 R20 相关路径合入 main）仅跑默认门，供 runbook 首条实填粘贴 Run URL。全量门禁仍为 **F-G validation**。