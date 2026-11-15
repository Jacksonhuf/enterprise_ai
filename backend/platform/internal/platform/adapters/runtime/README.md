# 运行适配层

本目录实际复用固定版本 WeKnora `AgentEngine`、`ToolRegistry` 和 Chat Completions 协议客户端。

- registry.go：服务端模型注册、环境变量凭据引用、连接测试；不会向浏览器返回秘密。
- engine.go：每轮发布快照、对话历史、受控服务工具注册、服务器端身份/版本/关联绑定。
- decode.go：受控工具参数严格解析。
- *_test.go：本地模型HTTP协议夹具、原ReAct多轮调用、身份注入/未引用服务拒绝、取消与秘密不泄露验证。

默认验证模式继续由应用API中的确定性场景调度处理；选择注册模型后进入此原引擎适配。原生产Session服务与其他工具尚未整合。源码导入需包含backend/platform/docs与website-docs，这两个包是上游原引擎的编译依赖。
