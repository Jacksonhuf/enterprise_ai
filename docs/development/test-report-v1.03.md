# 第103轮（v1.03）：R18 抽页回归与 R02 进度快照

## 变更摘要

- **浏览器**：`scripts-smoke.mjs` 增加 R18 抽页（`/settings/tenant`、`/governance/operations?tab=settings`、`/settings/models`、390px `/apps`）。
- **文档**：`progress-snapshot-2026-11-18-r02.md`、`mysql-schema8-r21-engineering-pilot-ledger-2026-11-18.md`；checklist 彩排子表。

## 测试

```bash
# MySQL 默认门
cd backend/platform && export COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race ./internal/platform/api/ -run TestR20Schema8RunbookDefaultSmoke

# 浏览器（需已构建 cognara-dev 与前端 production build，见 CI validation）
cd frontend && npm run test:cognara-browser
```