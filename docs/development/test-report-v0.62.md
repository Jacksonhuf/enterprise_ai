# 第62轮：R05 收口 — apply 后重启与多连接/API 等价

基线 main PR59。专项验证 production 路径 `MigrateMySQLAudit` → `PlanMySQLApplications` → `ApplyMySQLApplications` 之后：

- 关闭连接池再 `OpenMySQL`：应用/会话/检查逻辑状态与 apply 前一致（`equalApplicationJSON` 子集）。
- 两个 `OpenMySQL` 连接并发 `TrySaveApplicationDraftV3`：一成功、一 `ErrRelationalDraftConflict`。
- 编译态 `cognara-dev` 在 schema2 只读 GET 应用/发布列表 → stop → apply → 重启：HTTP JSON 与 apply 前一致。
- API 包：schema3 下双 `httptest` 主机并发会话重命名，一 200 一 409。

完整 race 套件与 `validate-contract-capture.py` 见 PR。R05 代码能力视为已齐；生产库运维切换与 R20 全量多实例仍另验收。