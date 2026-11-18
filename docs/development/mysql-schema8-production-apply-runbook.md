# MySQL schema8 生产 apply 运维登记（模板）

版本：0.1；与代码夹具 `PrepareSchema8IntegrationDB` / 专项测试顺序一致。  
**凭据不进命令行**：使用环境变量 `COGNARA_MYSQL_DSN`（或部署系统注入的同等配置）。

## 1. 前置条件

| 项 | 要求 |
|----|------|
| 起始版本 | `cognara_fg_state.schema_version = 1`，审计可迁移到 2 |
| 备份 | 全库逻辑备份 + 校验和登记（时间、操作人、备份路径） |
| 窗口 | 双实例（R20）计划内只读或灰度一台写 |
| 回退 | 保留 apply 前快照文件（schema 对应版本 export）或备份还原路径 |

## 2. 迁移梯（每步：plan → apply → 验证）

以下 CLI 在 `backend/platform` 构建产物中；每一步 **必须先 `--check`**，记录 JSON 中的 `source_sha256`，再 **`--apply <sha256>`**。

| 序 | 目标 schema | 命令 | apply 后 `schema_version` |
|----|-------------|------|---------------------------|
| 0 | 2（审计表） | `cognara-migrate-audit --check` / `--apply` | 2 |
| 1 | 3 | `cognara-migrate-applications --check` / `--apply` | 3 |
| 2 | 4 | `cognara-migrate-grants --check` / `--apply` | 4 |
| 3 | 5 | `cognara-migrate-sessions --check` / `--apply` | 5 |
| 4 | 6 | `cognara-migrate-settings --check` / `--apply` | 6 |
| 5 | 7 | `cognara-migrate-exceptions --check` / `--apply` | 7 |
| 6 | 8 | `cognara-migrate-usage --check` / `--apply` | 8 |

验证（每步后）：

```sql
SELECT schema_version FROM cognara_fg_state WHERE id=1;
```

应用进程：**全部实例滚动重启**，确认 `OpenMySQL` 可读且 `GET /ready` 为 UP（双实例各测一次）。

## 3. 登记项（运维填表）

| 字段 | 示例 |
|------|------|
| 执行时间（北京时间） | |
| DSN 库名（无密码） | |
| 每步 plan SHA256 | |
| apply 退出码 | |
| 回退决策 | 无 / 快照路径 / 备份还原 |

**填写样例（验证环境彩排，非真实生产库名）：**

| 字段 | 样例值 |
|------|--------|
| 执行时间（北京时间） | 2026-11-18 14:30 CST |
| DSN 库名（无密码） | `cognara_prod_fg` @ `mysql.internal:3306` |
| 每步 plan SHA256 | audit:`a1…` ; apps:`b2…` ; … ; usage:`h8…` |
| apply 退出码 | 全部 `0` |
| 回退决策 | 快照 `/backup/cognara-fg-v8-20261118.json` + SHA256 留存 |

## 4. 发布后烟雾（与 R20 测试对齐）

在 **两台** Go 实例共享同一 DSN 时至少执行：

- `GET /ready`（并发探测已登记，见 test-report-v0.83）
- 双主机会话标题 CAS、grant 并发、settings CAS（见 test-report-v0.81–v0.82）
- 治理 `usage` / `grants` / `exceptions` / `audit` 只读并发（见 test-report-v0.84–v0.87）
- 发布检查/发布 CAS、模板复制、grant 授撤交叉（见 test-report-v0.88–v0.89）
- 会话创建并发与跨主机续聊探测（见 test-report-v0.90）
- 治理关系化 `EXPLAIN` 烟测（见 `mysql-schema8-slow-query-checklist.md`）

自动化参考：

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/api/ -run R20Schema8
```

## 5. schema8 快照回退演练（变更单模板）

与 `TestR20Schema8RunbookSnapshotImportDrill` / `TestMySQLV8SnapshotRoundTripPreservesUsageRows` 对齐。

| 步 | 动作 | 登记 |
|----|------|------|
| 1 | apply 完成且 `schema_version=8` 的库上执行 export（校验 `source_schema_version` 与文件 SHA256） | export 路径、SHA256、操作人 |
| 2 | 在 **隔离** 目标库（非生产）上做 import，校验报告与版本号 | 目标 DSN（无密码）、import 退出码 |
| 3 | 双 `OpenMySQL` 连接读同一 usage 行数；`GET /ready` UP | 行数、ready 截图或日志 |
| 4 | 生产回退决策：仅当团队批准，在同一变更单内用 **备份还原优先**；import 作为演练或灾备副本重建 | 批准人、RPO 说明 |

自动化彩排：

```bash
cd backend/platform
export COGNARA_TEST_MYSQL_DSN='...' COGNARA_REQUIRE_MYSQL_TESTS=1
go test -p 1 -race -count=1 ./internal/platform/applications/ -run R20Schema8RunbookSnapshotImportDrill
```

## 6. 未包含

- RPO/RTO 实测（待 R21 与团队定标）
- 全量压力与 SLO 对照
- 生产库上的真实 import（须经变更单与备份门禁）