# Cognara · enterprise_ai

企业认知平台 F/G 开发仓库。采用 Vue 3 / TypeScript / TDesign 前端与 Go 后端，通过固定版本脚本导入 WeKnora 源码，在功能目录内扩展应用、治理和业务服务适配。

## 本次可运行范围

- 应用列表、空白/模板创建、四组配置、保存草稿、发布检查、不可变发布历史。
- 发布应用创建会话、供应链场景对话、结构化结果、来源证据、当前应用历史对话。
- 资源授权新增/撤销、企业身份视图、真实审计记录、基础用量、服务状态。
- Cognara 蓝色视觉主题、可折叠侧栏、左下账户、按角色显示入口。
- 资产与服务授权目录；配置页的只读连接状态及个人非敏感偏好保存。

**这是本地契约验证版本，尚不是完整产品。** A–E 使用独立 HTTP Mock；开发主程序默认采用确定性场景调度；配置模型后，已接入 WeKnora ReAct Agent 引擎与原协议客户端。原生产 Session 服务、SSO与数据库服务尚未完成整合。上游主服务、前端组件与源码通过导入脚本保留在功能目录中，供后续逐步移植。不能把导入源码等同于所有功能已经上线。

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
python -m pip install 'jsonschema[format-nongpl]==4.26.0'
COGNARA_CONTRACT_CAPTURE=/tmp/cognara-contract.jsonl go test -C backend/platform -count=1 ./internal/platform/api
python scripts/validate-contract-capture.py /tmp/cognara-contract.jsonl
```

`make check`检查本次新增模块和当前前端入口；不代表整个上游测试套件或原主服务集成均通过。详见 [开发状态](docs/development/implementation-status.md) 和 [来源与复用记录](docs/provenance/upstream.md)。

上游参考文件不重复提交，`.gitignore`只跟踪产品扩展与固定依赖清单。若需要修改导入文件，应显式使用`git add -f 路径`将该文件纳入版本管理，并在来源记录中说明补丁。不要仅修改未跟踪的参考文件。

## 第二轮：模型引擎与 SQLite

已接入固定版本 WeKnora `AgentEngine` / `ToolRegistry` / OpenAI-compatible Chat Completions 客户端；不是另写一个模型循环。只注册当前发布允许的受控服务工具，不开放系统命令、任意URL、SQL或Python沙箱。

模型配置示例见 `docs/development/models.example.json`。复制后填写自己的网关地址与模型名，密钥只设置在该配置引用的环境变量中；不要把密钥写进 JSON 或提交仓库。启动示例：

```bash
cd backend/platform
go run ./cmd/cognara-dev --demo --model-config ../../docs/development/models.local.json
```

模型配置页可查看注册状态及测试连接。随后在应用配置中选择主模型、保存并重新发布。未配置时继续使用 `validation-only`。测试连接会发起一次真实模型请求；协议自动化测试使用本地模型响应夹具，不消耗外部模型额度。当前没有真实提供方凭据，尚未完成商业模型服务实测。

可选 SQLite 存储（原 JSON 默认不变）：

```bash
cd backend/platform
go run ./cmd/cognara-dev --demo --storage sqlite --data .devdata/state.db
```

SQLite 使用 WAL、即时事务与每次读取重新加载状态，支持独立连接间的授权/配置可见性及事务回滚。这是本地持久化适配，不代表 PostgreSQL、多租户生产身份或多节点部署已经完成。切换存储不会自动迁移原 JSON 文件；请保留旧文件并显式选择原适配器读取旧数据。

受限内存环境的竞态检查采用 `-tags=nomsgpack -p 1`，关闭当前 JSON API 不使用的 Gin MessagePack 编解码，减少编译峰值内存。

SQLite 适配需启用 CGO 并安装 C 编译器；默认 JSON 适配不依赖 SQLite 服务。完整验证记录见 [第二轮测试记录](docs/development/test-report-v0.2.md)。

## 第三轮：受控资产与正式指标

原ReAct引擎按应用发布配置注册四种受控工具：资产发现、上下文读取、正式指标执行及服务调用。发现结果同时受发布引用与当前授权限制；上下文凭据不提供给模型。正式指标执行由服务端获取 `EXECUTION_CONTEXT`，再使用绑定身份、应用、发布、轮次及当前版本的凭据发起调用。

当前指标验证集实现 `chip-gap-metric` 的 `BASIC_METRIC`，结果与补货场景同口径；不支持任意SQL、用户指定版本、URL或未实现规则。现有数据不会自动扩大授权：要在既有应用中启用指标，管理员需显式授予该指标的 `asset.context.read` 和 `execution.invoke`，开发者选择资产与“正式指标执行”后重新发布。新验证数据库已提供这两条独立授权。

指标证据按指标权限重新鉴权，不需要额外授予场景服务调用权。消息保存事务内再次检查应用与本轮所用资源的授权。业务执行记录仍由A–E负责；Mock的上下文/幂等/执行记录只保存在当前进程内，不能代表跨重启恢复保证。

## 第四轮：历史恢复、请求追踪与异常管理

已完成助手消息可从 JSON/SQLite 中重建终止消息流；恢复不会调用模型或重做业务执行。历史读取、会话详情、重放及发送给模型的历史上下文统一应用当前资源授权，撤权内容会显示“访问受限”。新消息保存实际使用的资源与权限配置版本；旧消息按执行来源/原发布保守鉴权，未知配置版本不直接展示内容。

错误响应返回的请求编号与响应头一致，可在管理员异常管理中定位。异常支持确认、标记解决和重新打开，并使用版本号避免覆盖他人的处理。异常状态变更不会自动重试业务执行；错误正文不透传提供方原始诊断、密钥或SQL详情。

事件接收校验v0.3审计/用量字段；事件、去重凭据在同一次持久化更新中写入，补投可返回DUPLICATE而不重复入账。授权变更和发布也与对应审计记录同事务写入。审计、用量和异常均按企业隔离。

恢复保证仅覆盖已经持久化完成的消息。进行中执行恢复、A–E发送方发件箱、生产身份与分布式部署仍待整合。A–E原始证据由业务服务负责保存；Mock重启后仍可能无法读取旧执行证据。

接口校验需要日期时间格式依赖；校验器在缺少该依赖时会直接失败，避免静默跳过。对话输入上限按4000个Unicode码点检查，中文按字符计数。
