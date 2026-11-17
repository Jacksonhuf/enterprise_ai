# 第63轮：R06 授权拆表准备（DDL + 只读 plan）

基线 main PR60。新增 `cognara_fg_grants` DDL（`migrations/003_grants.sql`）、结构校验、`splitGrantLayout`/`joinGrantLayout`/`verifyGrantLayout`、只读 `PlanMySQLGrants`（要求 schema3 + 应用关系表，目标 schema4，拒绝重复 grant_id/重复有效元组/保留表已存在）及 `cognara-migrate-grants --check`。内部 `rehearseMySQLGrantMigration` 复制后回滚。

真实 MySQL：schema3 fixture 上 plan 与 round_trip_verified；schema2 拒绝；彩排不落库；损坏授权元组拒绝。无 apply、无运行时读写在表。完整 CI 见 PR。