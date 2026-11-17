# 第61轮：R05-D 显式 apply 与 schema3 运行时开放

基线 PR58（main `c42d747`）。新增 `ApplyMySQLApplications`：`PlanMySQLApplications` 校验与 checksum 匹配后，在可串行化事务内执行 `stageMySQLApplicationRoot`（含关系拷贝与根瘦身）并原子 `schema_version=3`，提交前 `readMySQLApplicationV3` 对账。`cognara-migrate-applications` 支持 `--apply --expect-sha256`（须先 `--check`）。`OpenMySQL` 在 schema3 且关系 schema 与对账通过时允许启动（无效手工 version 切换仍拒绝）。

真实 MySQL 8.4：`TestApplyMySQLApplicationsCommitsSchema3`（plan→apply→再开库→导出）、checksum 失败拒绝、手工 `schema_version=3` 无表拒绝；既有 v3 读/写/快照测试更新为合法 handoff 下可启动。完整 race 套件与 `validate-contract-capture.py`（199 对）通过。CI 见 PR。