# 第59轮：R05-B应用/检查/发布HTTP入口schema3适配

基线PR56。专用关系writer（创建、草稿CAS、检查、发布）此前未接业务入口，schema3下走通用Store会因应用域守卫失败。本轮在Store层增加`Try*`探测并在四条认知应用HTTP处理函数中优先走专用writer；schema1/2仍走原`UpdateOperational`路径。创建在schema3下不再重复`event`审计（关系创建writer已写`application.create`）；草稿保存补齐client/request审计元数据。

真实MySQL 8.4专项（本地8.4.9实例）：经Store完整走通空白创建→草稿保存（含审计元数据）→模板复制与revision拒绝→发布检查→发布消耗检查；通用Store改应用仍拒绝；并发草稿保存CAS；审计触发器故障时创建整体回滚。关系错误映射为既有409/404语义。启动仍拒绝schema3。

R05-B应用写入适配至此收口；C无损快照与D显式apply未完成，R05保持开发中。完整CI见PR。