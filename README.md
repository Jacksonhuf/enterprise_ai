# Cognara · enterprise_ai

企业认知平台 F/G 开发仓库。采用 Vue 3 / TypeScript / TDesign 前端与 Go 后端，通过固定版本脚本导入 WeKnora 源码，在功能目录内扩展应用、治理和业务服务适配。

## 本次可运行范围

- 应用列表、空白/模板创建、四组配置、保存草稿、发布检查、不可变发布历史。
- 发布应用创建会话、供应链场景对话、结构化结果、来源证据、当前应用历史对话。
- 资源授权新增/撤销、企业身份视图、真实审计记录、基础用量、服务状态。
- Cognara 蓝色视觉主题、可折叠侧栏、左下账户、按角色显示入口。
- 资产与服务授权目录；配置页的只读连接状态及个人非敏感偏好保存。

**这是本地契约验证版本，尚不是完整产品。** A–E 使用独立 HTTP Mock；开发主程序采用确定性场景调度，不调用大模型，也尚未接入 WeKnora 生产 Agent/Session 引擎。上游主服务、前端组件与源码通过导入脚本保留在功能目录中，供后续逐步移植。不能把导入源码等同于所有功能已经上线。

## 启动

要求 Go 1.26.0+、Node.js 24、npm、Python 3（仅用于开发导入脚本）、git。首次需联网获取固定提交，先运行`make bootstrap`；源码展开到功能目录，已有产品文件不会被覆盖。两个终端运行：

```bash
make dev-api
make dev-ui
```

访问 `http://127.0.0.1:5173`。API 8080，A–E Mock 8081，均只监听本机。API 启动必须显式提供 `--demo`；不要用于生产或向公网暴露。

登录页选择普通用户、应用开发者、管理员或隔离验证账户。选择普通用户 → 渠道补货助手 → 进入应用 → 提问“SO1002需求增加到350后，有多少物料缺口？” → 查看依据与来源。

验证时点为 **2026-04-02**，目标350、成品可用20、待产330、芯片需求336.6、当前可用60、原始缺口276.6。未纳入替代料和后续到货，不构成交期承诺。来源可用量可能含在途，不代表当天实物库存。

A–E具备契约端点后，可用`--cognitive-url`指定目标并通过`COGNARA_SERVICE_TOKEN`提供服务凭据；此时不启动本地Mock。目标必须能回调Go内部身份/授权端点。这个开关只替换业务适配，浏览器身份仍为本地验证账户，不能据此用于生产。

开发数据保存在 `backend/platform/.devdata/state.json`；仅适用于单进程开发。不要放入真实客户数据或凭据。

## 目录

```text
frontend/src/cognara/          当前产品入口、页面、主题、请求与权限导航
frontend/src/components/       导入的通用UI组件
frontend/src/views/             上游页面移植参考；不直接开放为产品路由
backend/platform/cmd/cognara-dev/ 本机契约验证主程序
backend/platform/cmd/server/    导入的上游生产主程序；本轮未完成接线
backend/platform/internal/platform/
  applications/                草稿、发布快照、会话与本地事务存储
  api/                         前后端API、内部身份/授权/事件入口
  governance/                  本地身份、短期签名委托
  connectors/cognitive/        认知服务HTTP适配器
  mockae/                      不可用A–E的独立HTTP验证适配器
contracts/api/fg-ae-v0.3.json    当前18项内部接口契约
contracts/examples/            契约样例
scripts/                       真实HTTP契约验证
```

## 验证

```bash
make check
python -m pip install jsonschema==4.26.0
COGNARA_CONTRACT_CAPTURE=/tmp/cognara-contract.jsonl go test -C backend/platform -count=1 ./internal/platform/api
python scripts/validate-contract-capture.py /tmp/cognara-contract.jsonl
```

`make check`检查本次新增模块和当前前端入口；不代表整个上游测试套件或原主服务集成均通过。详见 [开发状态](docs/development/implementation-status.md) 和 [来源与复用记录](docs/provenance/upstream.md)。

上游参考文件不重复提交，`.gitignore`只跟踪产品扩展与固定依赖清单。若需要修改导入文件，应显式使用`git add -f 路径`将该文件纳入版本管理，并在来源记录中说明补丁。不要仅修改未跟踪的参考文件。
