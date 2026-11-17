# 来源与复用记录

WeKnora fork： https://github.com/Jacksonhuf/WeKnora
固定提交：`cbdcebabf48c5ee2d6868b370326aa8341703f62`。

前端导入到 frontend，Go 源码导入 backend/platform；不使用品牌作为顶层软件目录名。Go module 及原 import 路径暂保留，避免不必要地改写全部上游代码。LICENSE、THIRD_PARTY_NOTICES、嵌套第三方许可均保留。

当前活跃界面复用 `ResourceListToolbar.vue`、`ResourceCardAvatar.vue`、`EmptyState.vue`、通用图标及离线图标保护，沿用 Vue/Pinia/Vue Router/TDesign 与国际化基础。`src/upstream-main.ts`保留原入口；新入口是`src/main.ts`。本轮新增 Cognara 页面、契约HTTP适配和本地开发宿主。

原 Go 主服务、会话与租户服务尚未整体接入开发宿主。第二轮已实际接入原 AgentEngine、ToolRegistry、模型消息类型与 Chat Completions 协议客户端，并以端到端HTTP测试验证；其他服务仍不能视为已上线。

未启动或移植 Python DocReader 服务。保留 Go DocReader 客户端/协议供原 Go 源码引用；部分原 Go embed 所需 Python 文件或协议生成文件可能仍随源码存在，不能据此认为 Python 服务已启用。Python技能沙箱与原高级沙箱能力在新入口不可用。

供应链案例来源： https://github.com/sharptoolbox/Onto-SupplyChain ，历史验证基线 `9599214ca97152a2c7d1cfa2b7fabb114298aad8`。当前只提取SO1002场景的合成验证常量，不导入真实企业数据、不实现A–E专业服务。

## 仓库交付方式

`make bootstrap`调用` scripts/import-source.py`按完整提交SHA获取源码，校验HEAD，再复制缺失文件到上述功能目录。开发扩展、前端入口、主题、依赖锁文件、契约与测试直接纳入当前仓库；上游参考源码不重复提交。该方式需要首次联网，不能当作完全离线源码包。固定版本许可随源码保留，核心许可证也直接提交。

第五轮持久化复用原Go模块已锁定的`github.com/jackc/pgx/v5`驱动，不另引入数据库客户端。新的PostgreSQL适配与原有验证聚合接口兼容；并不表示已经迁移到上游生产Session/用户表。

### TDesign RadioGroup 1.19.2生命周期兼容补丁

2026-11-17：浏览器CI捕获TDesign `es/radio/group.mjs` calcBarStyle在卸载后nextTick/ResizeObserver回调读取空DOM引用。通过frontend/scripts-tdesign-lifecycle.mjs的Vite pre transform局部添加disposed/DOM连接检查及隐藏测量临时节点清理，保持原组件/视觉和MIT声明，不修改依赖目录。版本或源码锚点变化时构建失败并要求重新检查；测试直接验证已安装上游测量代码原始失败与补丁行为。升级TDesign时检查上游修复并移除此临时补丁。
