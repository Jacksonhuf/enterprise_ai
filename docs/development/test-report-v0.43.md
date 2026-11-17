# 第43轮：任务计划与公共配置一致性

日期：2026-11-17。main基线：PR39合并提交 `a6603ab970a8d656ca302cb515137c30b349f1e8`。

## 目标与实现

建立 [剩余任务与开发测试计划](remaining-work-and-delivery-plan.md)：22项任务、六阶段、逐项验收与外部依赖，后续优先推进MySQL关系模型。

R01修复：`GET /api/v1/cognitive/governance/settings` 从同一租户范围的public设置读取已保存公告，复用现有MySQL设置投影和失败传播；移除固定语言/时区。保留items/name/value和validation响应形态，无新API/权限/数据库结构。运营公共配置增加到现有编辑页的入口；公告按Vue文本展示。未提供全用户公告分发。

新增两个Go顶层测试：保存→管理列表→重启读取→旧版本冲突→清空→权限拒绝/审计数量；直接验证两个企业管理员相同用户ID的命名空间及旧格式隔离。扩展已有真实MySQLschema1/2设置测试核对管理列表，并扩展数据库关闭503测试。浏览器脚本新增实际表单保存→运营Tab→刷新，并断言含HTML的公告只显示文本，公告为demo-user时也不能被通用身份标签替换。

## 本地验证

- 前端97项测试、专用vue-tsc、生产构建与包预算通过；入口248330字节、总JS753866字节。
- Go平台及四个命令包竞态全回归重跑通过，共136个顶层测试定义；本地未配置MySQL/PostgreSQL，相关用例跳过，不把跳过算真实数据库通过。首次编译发生标准库importcfg读取错误，源码编辑完成后重跑通过；不将该首次失败隐藏。
- Go开发宿主构建、JSON及SQLite实际启动/业务调用/重启与持久化回归通过。
- 本地Chromium在启动时被执行环境拒绝Unix socket（Operation not permitted），未进入页面；不宣称本地浏览器通过，使用CI浏览器作验收。
- `git diff --check`通过。

## 集成门禁与边界

完整CI必须覆盖Go竞态、内部HTTP契约捕获、真实MySQL/PostgreSQL、四存储运行与恢复迁移、前端/类型/构建预算及Chromium。精确提交、运行链接和最终结果记录于[PR #40](https://github.com/Jacksonhuf/enterprise_ai/pull/40)；未通过不能关闭R01。R03的querySelector偶发错误根因仍未确认，现有pageerror诊断及20次SPA往返不移除。

未调用商业模型、真实身份提供方或A–E真实服务。不改变演示身份和当前受控能力边界；不以本轮小修推高整体生产完成度。
