# Platform API：F / G

本目录是 enterprise_ai 中 F 应用构建与运行、G 横向治理的 Go 后端位置，对应部署单元 platform-api。

当前仅创建目录和职责说明；尚未导入 WeKnora 代码，未创建 go.mod，也不是可运行服务。

## 目录职责

- internal/platform/applications：F1/F2/F3，标准应用、模板副本、四组配置、不可变发布、会话绑定及 Turn 引用。
- internal/platform/governance/authorization：G1 资源授权，默认拒绝。
- internal/platform/governance/delegation：G1 可信用户与应用/Agent上下文、服务间委托。
- internal/platform/governance/audit：G2 关键事件、幂等去重及原审计适配。
- internal/platform/governance/operations：G4 基础配置、状态、异常和基础运营。
- internal/platform/connectors/cognitive：调用 A/D/E 的 HTTP 客户端，不实现 A—E。
- internal/platform/adapters/runtime：适配已有 Agent、Session、身份与审计。
- cmd/server、internal/router、handler、container、middleware、types、agent/tools：预留上游复用接入位置。
- migrations：仅本服务拥有的数据迁移。

G3后置，不建立空服务。F/G共用一个Go后端；前端独立位于仓库根目录 frontend/。

## 后续实施约束

复用已核查上游运行链路，不再实现第二套模型、消息或会话系统。先确认上游 Go module 与许可证，再导入源码；目录功能命名不要求立即改写上游 import。

应用保存不等于发布；发布不等于授权。每轮记录实际应用发布版本，已有 Session 下一轮使用最新发布。D3证据和G2审计分别管理。不得跨库直接操作 A—E 的数据表。

依据：FG_Service_and_Repository_Architecture_v1.1、FG_AE_Interaction_Contracts_v0.2、Enterprise_AI_WeKnora_UI_Standard_v1.0。
