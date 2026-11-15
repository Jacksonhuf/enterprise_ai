# 平台统一前端：F/G目录初始化

全平台采用一个 Vue 3 / TypeScript / TDesign 前端，位于根目录 frontend/，独立构建部署为 platform-web。

本次仅初始化 F/G 负责的页面与组件位置，未导入上游源码，没有 package.json 或可运行页面，不创建 A—E 专业页面。

- src/views/cognitive-apps：F应用入口、配置、详情和必要发布展示。
- src/views/chat：复用现有对话页面的目标位置；避免新建重复聊天系统。
- src/views/governance/access、audit、operations：G1/G2/G4页面。
- src/components/cognitive：F使用的资产/服务选择及结果/证据展示适配。
- src/api、stores、router：F/G后续接入现有前端体系的位置；不是新的前端框架。

UI依据 Enterprise_AI_WeKnora_UI_Standard_v1.0；物理位置以目录方案V1.1为准。四空间、左下唯一账户、四组应用配置、按应用隔离会话。资产/服务选择分别对接A4/E3权威接口，不在前端复制权限权威。
