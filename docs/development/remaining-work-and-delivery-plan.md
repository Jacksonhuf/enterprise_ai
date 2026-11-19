# F/G 剩余任务与开发测试计划

版本：1.2；日期：2026-11-17（北京时间）。本轮代码基线：main 合并 PR #47，提交 `c098aac97e2628defbe5e0f6b49bdcb6ea71bdcb`。本文件是后续迭代的任务入口；每轮同步状态、测试证据和下一项任务。

## 1. 范围与现状

只负责统一前端中的应用构建、应用使用、横向治理，以及 `backend/platform/` Go 服务。A–E 专业建模、图谱、数据执行与语义服务由对应团队负责；这里承担调用适配、权限协作、契约和验收，不替代他们开发。统一前端 + Go 平台服务 + A–E 认知服务的交付边界不变。不新增 F/G 独立前端、独立 Runtime 或模型网关。

依据：当前《FG_Service_and_Repository_Architecture_v1.2》《FG_API_Inventory_and_Parameters_v0.3》《FG_AE_Interaction_Contracts_v0.3》《Enterprise_Cognitive_Platform_Page_Inventory_v1.2》与现有仓库。设计中的候选页面不自动成为必须独立建设的路由；129 个 Figma 帧不是129个正式页面。实际 UI 不展示架构字母编号。

已有：应用/模板、草稿 CAS、不可变发布与历史恢复为草稿、受控 Go Agent 工具、聊天和 SSE、完成结果重放、授权/撤销、审计、异常状态处理、基础用量、个人偏好、模型适配和存储迁移验证。核心业务目前主要由合成供应链数据及 Mock A–E 验证。用户组/角色和浏览器身份仍是验证实现；部分连接配置仅有只读占位。

MySQL 是生产目标；目前仍以聚合状态 JSON 为主，审计已拆表，部分读取/更新有投影优化。这不等于完整关系模型或多实例生产就绪。PostgreSQL/SQLite/JSON 保留为兼容及验证适配器，不改变目标选型。

## 2. 状态与完成口径

- 待开发：当前尚未满足该项验收；已有局部实现不能替代整项完成。
- 开发中：有本轮实现，但相关测试或集成验收尚未完成。
- 待外部联调：可先完成适配与 Mock 回归；没有真实提供方时保持未验收。
- 实现完成：代码与本地专项已交付，完整集成门禁的实时状态跟踪关联PR；尚不自动计为已验收。
- 已验收：实现、相应自动测试及证据齐备；标明范围，生产验收另计。
- 后置：有明确范围决策才进入，不计入首期必交项。

不按代码行、测试数、页面帧数推导百分比。此前“完整功能65%–70%、Mock闭环约80%、生产准备约55%”仅为历史工程估计，不是本计划的加权验收率。后续报告采用任务状态、阶段门和阻塞项；不得把 A–E 完成度外推为整个平台进度。任务粒度不同，不以“完成项/总项”冒充产品完成百分比。

## 3. 剩余任务台账

P0：影响生产正确性或安全；P1：首期完整体验及可运营性；P2：启用后才需要的扩展。工作量为单人有效开发测试工作日粗估，不含外部等待、业务评审或新增范围。每个大项可以拆多个 PR，验收条件全部满足才关闭。

| ID | 优先级 | 剩余任务/当前缺口 | 验收标准 | 前置/责任 | 粗估 | 当前状态 |
|---|---|---|---|---|---:|---|
| R01 | P1 | 公共配置列表仍返回固定值 | 使用已保存的本企业公告；保存/刷新/重启/清空一致；旧版本冲突不覆盖；无权拒绝；HTML按文本展示；数据库失败不伪报成功 | F/G，现有settings接口 | 1 | 专项已验收：PR #40最终head完整CI通过；不含全用户公告投放 |
| R02 | P1 | 完整页面与API验收映射 | 对本节后的每个F/G设计条目、启用配置页及既有API登记源码、测试、截图或缺口；路由/Tab/Drawer不重复计数；逐项给出Mock/真实状态 | F/G，冻结现有范围 | 2–3 | 部分实现：MySQL/schema8 API 证据索引 v0.85；全量截图/字段仍待开发 |
| R03 | P0 | 偶发UI生命周期异常根因未确认 | 保留pageerror失败门禁；复现时记录调用栈并修复；抽屉/Tab/后退/快速导航回归；不能仅因一次通过宣称根治 | F/G；PR39诊断保留 | 1–3 | 部分完成：PR40修复RadioGroup卸载后测量且最终CI通过；其他UI路径仍待R02/R18验收 |
| R04 | P0 | MySQL业务关系模型与迁移设计 | 应用、草稿、发布、授权、会话、消息、异常、用量、设置逐表说明租户键/索引/CAS/事务；明确审计原子性、导入导出、旧格式升级、回退策略；兼容原API | F/G；现有schema1/2 | 2–3 | 设计完成：见关系模型设计；后续实施与迁移门禁分别验收 |
| R05 | P0 | 应用/草稿/发布/检查拆表 | 真实MySQL升级和重启；不可变快照、检查版本绑定、并发草稿409、发布原子性与租户隔离；旧快照迁移前后API等价 | R04，F/G | 3–5 | 实现完成（代码）：A–D与apply后重启/双连接CAS/编译态GET等价已专项；整项待生产运维apply与R20多实例门禁登记已验收 |
| R06 | P0 | 授权与租户范围拆表 | 主体/资源/操作/范围索引；创建去重、撤销幂等；并发授权与撤销正确；事务失败无部分审计/授权 | R04，F/G | 2–4 | 实现完成（代码）：schema4 迁移/读/写/快照；R20 多实例并发另验 |
| R07 | P0 | 会话/消息/Turn拆表 | 按租户/应用/用户/会话分页；重启完成结果重放；撤权后脱敏；并发写无丢失；迁移对账 | R04，F/G | 3–5 | 实现完成（代码）：schema5 迁移/读/写/快照；schema8 治理 usage 分页与撤权脱敏 MySQL 验收（v0.81） |
| R08 | P1 | 异常/用量/设置拆表及锁收敛 | 筛选分页、异常CAS、配置CAS、用量去重等价；移除对应聚合整段更新；真实MySQL多进程并发及慢查询证据 | R04–R07，F/G | 3–5 | 实现完成（代码）：settings/exceptions/usage schema6–8 迁移/读/写/快照；R20 并发与生产 apply 另验 |
| R09 | P0 | 共享生产身份方案与接入点 | 与团队明确SSO/JWT发行方、audience、租户/角色映射、撤销及密钥轮换；验证身份只能显式demo启用；缺失配置默认拒绝 | F/G实现端口；共享身份团队提供约定 | 2–3 | 待开发，提供方待对齐 |
| R10 | P0 | 真实登录/用户/组/角色接入 | 使用共享身份源；过期/错aud/跨租户/撤销拒绝；组变化影响授权；UI不把演示固定列表当真实身份；退出清理 | R09，身份团队+F/G | 3–5 | 待外部联调；可先适配Mock |
| R11 | P0 | A–E契约与输出Schema冻结 | 逐操作核对当前契约、错误码、分页、版本、幂等、超时与输出字段；拒绝未知能力；变更需版本化；维护HTTP捕获验证 | F/G+对应A–E负责人 | 2–3 | 待开发；需联合确认 |
| R12 | P0 | 真实A–E调用与权限链验收 | 目录→配置引用→发布→会话→执行→证据追溯完整真实链；双身份和权限/范围；403/撤权/退役/超时/不可用；不用Mock替代真实通过 | R09、R11；A–E可用端点与验证凭据 | 3–5 | 待外部联调 |
| R13 | P0 | 关键事件持久待发送与有限重试 | 业务提交与outbox同事务；幂等事件ID；可恢复重试、退避上限、失败可查；成功业务不因投递失败伪报回滚；接收方回执区分本地入库 | R04及R11；F/G发送端，A–E接收端协作 | 3–5 | 待开发 |
| R14 | P0 | 运行中断恢复与取消语义 | Turn状态持久化；进程中断后可识别未完成状态；明确续执行/失败恢复策略，禁止盲重放有副作用调用；停止接收与远端取消区分；重复取消幂等 | R07、R11；复用Go Runtime | 3–5 | 待开发；远端取消需提供方支持 |
| R15 | P0 | 上游Go主宿主集成验证 | 将当前平台适配器接入正式容器/Session/消息/路由装配；必要编译和回归通过；无效应用上下文不降级普通问答；不复制第二套Runtime | R05、R07、R09，F/G | 3–5 | 待开发 |
| R16 | P0 | 真实模型与秘密配置验收 | 配置一个选定商业模型的真实凭据引用；连接/流式/取消/超时/限流/失败真实测试；秘密不返回前端与日志；供应商用量缺失时明确未知 | F/G；有效模型端点/凭据 | 2–3 | 待外部联调；协议测试已有 |
| R17 | P1 | 启用能力的Go组件接入 | 逐项选择MCP/网页搜索/文档读取/存储/向量/队列/Skills中首期实际需要项；复用Go组件；有权才执行、参数校验、超时、脱敏、真实连接；未启用保持明确禁用 | R09、R16；每个提供方单独验收 | 每项2–4 | 待范围确认，不能承诺全部移植 |
| R18 | P1 | 全页面视觉与交互验收 | 逐页对照当前Figma/UI规范；角色导航、蓝色主题、字体/间距/密度、左下账户、无架构编号；空/错/加载/拒绝/窄屏/键盘；截图差异闭环 | R02，F/G；当前设计文件 | 3–5 | 待开发；代表屏已有实现 |
| R19 | P1 | 供应链验证数据扩展 | 增加无缺口/多物料/无权限/退役版本/缺数据等可重复夹具；记录来源、脱敏与预期值；规则计算正确，模型不编数字；真实数据与合成数据分开 | R11；真实数据依赖业务/A–E | 2–3 | 待开发；SO1002夹具已有 |
| R20 | P0 | 多实例、容量与故障测试 | 两个Go实例共享MySQL；并发CAS/幂等/撤销一致；压力、连接池、SSE中断和依赖故障；记录测量值，与团队约定SLO对照；未约定不编造达标 | R05–R08、R13–R16 | 3–5 | 开发中：双 HTTP 矩阵 + continue-stream 重放 + /ready 轻量探测；正式压力/SSE断连/SLO 另验 |
| R21 | P0 | 生产部署/备份/恢复/可观测性 | 常规容器+入口/MySQL方案；健康/就绪、指标、日志关联、配置校验；数据库备份还原演练、升级失败演练、运维说明；RPO/RTO待团队定标后实测 | R08、R13、R20；运维协作 | 3–5 | 待开发；Cloudflare方案继续暂停 |
| R22 | P0 | 联合UAT与发布关闭 | 业务用户/应用开发者/管理员真实端到端验收；安全隔离、性能和恢复证据；P0缺陷清零；已知限制与回退版本登记；完整范围才称生产可用 | R10、R12、R15–R21 | 2–3 | 待开发，依赖联合环境 |

G3跨域依赖影响治理、完整收费账单、完整国际化、停用/撤回/直接线上回滚、Python技能沙箱不在当前默认首期范围。需要时先补需求和契约；“历史恢复为草稿”不代表已有线上回滚。企业公告本轮只供管理页查看，不代表全用户公告分发。

## 4. 页面/API验收矩阵的建立范围（R02）

| 设计条目 | 当前实现入口 | 已有能力 | 尚需验收/补齐 |
|---|---|---|---|
| PAGE-001 / PAGE-002 | `/apps`；`/apps/:id/chat/:session?` | 已授权应用、聊天/历史/证据投影 | 真实模型/A–E、异常/窄屏/视觉全态 |
| PAGE-F2-01 / PAGE-F2-02 | `/build/apps`；`/build/apps/:id/edit` | 模板/空白创建、四组配置与CAS | 全字段设计核对、真实引用与提供方 |
| CAND-05 / CAND-06 / CAND-07 | `/build/apps`；`/build/apps/:id`及发布Tab | 管理、详情、发布历史、恢复为草稿 | 完整视觉交互；不默认补停用/撤回 |
| CAND-08 / CAND-09 / CAND-11 | `/governance/access?tab=users/groups/roles` | 演示列表及详情 | 真实身份源、成员/角色的只读或写入职责确认 |
| CAND-10 / CAND-12 | 同上`clients/grants` Tab | 应用身份、授权审阅、创建/撤销 | 真实主体、范围语义、多租户 |
| CAND-13 | `/governance/audit` | 筛选分页、详情、异常请求关联 | 真实跨服务事件送达、投递回执 |
| CAND-14 / CAND-15 / CAND-16 | `/governance/operations`各Tab | 状态、异常CAS、基础用量 | 真实采样、完整统计口径；用量不是账单 |
| CAND-17 | 同上`settings`；`/settings/public` | 公告保存；本轮补公共列表一致性 | 全页设计验收；不宣称公告分发 |
| Go支撑配置设计子视图 | `/settings/:section` | 模型目录/协议适配；部分明确禁用 | 按R17逐项启用，不能把只读占位算已实现 |
| PANEL/CMP及工程态 | 随所属页Drawer/Dialog；登录/拒绝/不存在 | 已有若干确认/详情与拒绝态 | 逐个挂靠业务条目验收；认证方式由共享身份决定 |

17个F/G主设计条目按页面清单中的2个门户PAGE、2个F2 PAGE、CAND-05—17统计，只是映射范围，不是17个独立路由；A–E的其他条目不由本计划代建。R02需进一步为每项挂实际源码、用例和截图，目前本表是范围矩阵，不能冒称完成了逐字段验收。API以现有OpenAPI和HTTP捕获验证为基准；不为计划另造接口数量。

## 5. 开发顺序与阶段门

| 阶段 | 任务 | 退出条件 | 排期方式 |
|---|---|---|---|
| S0 计划与基线 | R01–R03 | 配置问题闭环；覆盖矩阵形成；UI风险有证据与处理结论 | 本轮R01；后续先做R04，再在各页回归中推进R02/R03 |
| S1 生产数据基础 | R04–R08 | 真实MySQL迁移/并发/租户隔离/恢复通过，API行为兼容 | R04→R05→R06→R07→R08；每个小PR独立迁移门禁 |
| S2 可信接入 | R09–R12、R16 | 真实身份、模型、A–E业务调用至少一条端到端验收 | 先接口/适配器及故障夹具；外部依赖有条件再联调；不阻塞S1 |
| S3 可靠运行 | R13–R15 | 事件不丢、执行中断可识别与恢复、正式Go宿主可用 | S1数据基础先完成；按契约隔离外部取消能力 |
| S4 产品验收 | R17–R19 | 已启用能力真实可用；全部范围页面和供应链场景验收 | 随每轮增量进行；首期能力冻结后重新估时 |
| S5 生产发布 | R20–R22 | 联合安全/压力/故障/恢复/UAT全部达标 | 外部环境、模型和身份齐备后执行 |

以上是依赖顺序，不代表每阶段完全串行。当前粗估仅用于拆分，单人按完整首期范围推进约11–18周有效工作量（未含R17按项增加）；提供方数量、上游集成差异与迁移反馈可能改变估算，外部等待另计。多人大幅并行须先按数据/身份/契约/前端任务边界分工，不能直接将工期除人数。每轮完成后依据实际耗时重新估算，不承诺自动后台无限执行。

## 6. 每轮开发—测试循环

1. 从就绪队列取一个可闭环范围，记录任务ID、依赖、原问题与验收；先读已有Go/前端组件，优先复用。
2. 实现最小可审阅改动；保持当前UI/API。必要契约变更先更新规范与兼容方案；任何无真实服务的能力明确标注Mock/未连接。
3. 执行与风险匹配的专项测试；修复失败后重跑受影响测试。涉及持久化必须包含重启、迁移/事务或并发；涉及UI须包括真实生产构建浏览器流程。
4. 运行现有门禁：平台Go竞态回归、HTTP契约捕获校验、真实MySQL/兼容存储运行测试、前端单元测试、专用类型检查、构建及包预算、Chromium smoke。数据库不可用时本地跳过必须写明，由CI提供真实数据库证据。
5. 提交PR到main；记录准确head和CI结果。CI失败继续修复，未通过不得登记“已验收”；不得屏蔽非预期pageerror。合并前再次核对同一head检查及评审；已有授权范围内继续执行。
6. 更新本台账、当轮测试报告及进度快照，给出已完成/仍阻塞/下一项。Mock通过、真实联调、生产UAT分别记录。

测试数量只用于回归规模，不替代覆盖判断。当前PR39基线为134个Go顶层测试定义、97项前端测试及205组内部HTTP契约，完整CI证据见 [PR39](https://github.com/Jacksonhuf/enterprise_ai/pull/39)。UI的querySelector错误本轮再现并定位到TDesign RadioGroup卸载后异步DOM测量，已加版本受控补丁和4项确定性测试、8次抽屉/resize/SPA卸载回归。准确完整CI结果见PR40；R03其他UI路径与R18逐页验收继续开放。

## 7. 当前轮与下一轮

当前代码基线PR47；R01专项已验收、R04设计完成、R03部分修复、R05开发中。当前进度口径与证据统一见[进度评估](progress-snapshot-2026-11-17.md)。后续各轮增量为历史记录，不覆盖此处当前状态。

R05后续必须按以下顺序闭环，而不是继续将演练等同正式切换：

| 子项 | 下一步交付 | 退出检查 |
|---|---|---|
| R05-A | schema3一致性读取、State根+关系域重组、旧查询版本分支 | 不接受双权威、不默认空数据、不开放外部激活 |
| R05-B | 事务writer与旧定向写入适配 | 草稿CAS/检查绑定/发布不可变/租户隔离/同事务审计 |
| R05-C | 原文关系布局无损快照导入导出与恢复 | 隔离库重启/恢复、原文数值与扩展保真、失败无部分提交 |
| R05-D | 显式apply与版本原子切换 | A–C先通过；旧writer拒绝、API等价、多实例/重启/故障验证 |

A–D是R05拆分实施队列，不新增四个总体任务。v2恢复桥接保留兼容用途，不替代C。R05原3–5日粗估需重新校准；原总工期为历史工作量参考，不作为当前剩余承诺。

继续R06→R07→R08；R02/R03与页面回归同步。R09身份和R11真实A–E契约应开始对齐，外部环境缺失不妨碍内部适配，但不能登记真实联调已验收。

### 第46轮增量

R05新增内部SQL复制/投影核对事务演练；成功也回滚。尚未完成形态元数据持久化和原始JSON完整重组，公共命令继续只读。下一步完成持久化布局与无损重组后，才实施正式apply与schema3运行时切换。证据见test-report-v0.46.md及关联PR。

### 第47轮增量

PR43已合并。R05补原文LONGBLOB、根封套与checks形态持久化、独立数据库重载及严格源对账。验证范围包括隔离库提交/断开/重开、形态组合和损坏拒绝；运行时仍schema1/2，公共命令只读，不登记R05整项已验收。新增封套表属于内部迁移布局，正式接线必须避免剩余根聚合域出现双写权威。下一步正式apply前先完成schema3根归属、reader/writer及旧投影/快照接线，然后整体切换门禁。证据见test-report-v0.47.md与PR最终head CI。

### 第48轮增量

PR44已合并。R05内部演练接实际schema2源：事务锁定State屏障后重读版本和body、核对审计并复制/重组；不接受调用方旧快照，成功仍回滚。新增独立连接锁阻挡、最新源读取及中途故障回滚测试，证据见test-report-v0.48.md及PR。正式运行时/快照接线前仍不开apply，R05保持开发中。

### 第49轮增量

PR45已合并。R05内部关系布局→既有v2快照恢复桥接交付，新增独立空库导入/再导出及损坏/取消拒绝回归。v2目标使用原生JSON，数值归一化边界明确登记，不算schema3无损恢复已验收；原文关系布局的正式恢复仍待开发。见test-report-v0.49.md和关联PR最终CI。

### 第50轮增量

PR46已合并。R05确定剩余聚合域继续由State.body单一持有，新增移除apps/checks、清空封套根及State根+关系原文重组的内部回滚交接演练。拒绝重复根权威，验证根更新可见、成功/中途失败均完整回滚。设计第10节及test-report-v0.50.md登记边界。下一步接schema3正式读写、原投影与无损快照，全部通过再开放版本切换；当前schema2中间态禁止提交。

### 第51轮增量

PR48文档基线已合并。R05-A新增内部schema3一致性读取器（返回State及原文），验证隔离重开、可选审计、并发根更新的一致性快照、双权威拒绝和旧writer/普通启动继续阻断。A仍需全量旧定向读取版本接线，B–D未完成；不开放运行时schema3。证据见test-report-v0.51.md及PR最新head CI。

### 第52轮增量

PR49已合并。R05-A推进设置只读、异常/用量分页与审计查询的schema3接线；同事务验证关系状态、拒绝歧义别名，写入与普通启动继续关闭。专项验证v2/v3结果等价及损坏拒绝。A仍需全量入口/健康/快照核对；B–D尚未完成，禁止提前激活。schema3查询当前有完整关系校验成本，性能另验收。见test-report-v0.52.md及PR最终CI。

### 第53轮增量

PR50已合并。R05-A通用State与operational读取接schema3，Check复用关系验证；启动同快照只允许1/2，locking读取及旧审计/异常/设置writer仍拒绝3。旧快照导出明确拒绝且不产文件，待C实现。下一步进入B的事务writer及原文扩展保留，配合C无损快照后再考虑D激活；不因只读已接通开放schema3。证据见test-report-v0.53.md及PR最终CI。

### 第54轮增量

PR51已合并。R05-B首条内部草稿保存事务交付：租户+revision CAS、原文对象扩展保留、头/草稿/审计原子更新；验证双连接仅一成功、外租户拒绝、审计故障回滚与重开。数组显式变更整字段替换；仍保留State锁。下一步创建/检查/发布和通用业务writer接线，C/D未完成，不开放HTTP或schema3启动。见test-report-v0.54.md与PR最终CI。

### 第55轮增量

PR52已合并。R05-B新增内部应用创建事务，模板租户/预览版本绑定、原文复制与新身份、空白创建、原子审计、故障回滚/重开专项；无跨重试创建幂等承诺。下一步检查/发布及通用业务写入，C/D仍未完成，生产schema3未开放。见test-report-v0.55.md及PR最终CI。

### 第56轮增量

PR53已合并。R05-B新增内部发布检查与发布事务：检查绑定租户/操作人/锁定草稿revision/有效期；发布校验检查与当前发布CAS，以草稿原文追加不可变快照，同事务消费检查并写审计。验证同一检查并发仅一次发布、过期/旧草稿拒绝、审计故障整体回滚及重开。无跨重试幂等承诺；未接HTTP/启动。下一步其余业务writer（授权、会话、设置等仍经聚合State）与旧投影写入适配，再进入C/D。见test-report-v0.56.md及PR最终CI。

### 第57轮增量

PR54已合并。R05-B新增内部schema3根域通用writer：锁定State并经关系读取器重建，回调仅可修改会话、授权、设置、用量、回执与异常等根域；改动应用或检查即拒绝；审计沿用只追加/不可变规则；保留未知根字段。验证并发无丢失更新（含与专用草稿writer并发）、审计故障回滚及重开。未接HTTP/启动。下一步业务入口按域切换到关系/根writer的适配层，再进入C/D。见test-report-v0.57.md及PR最终CI。

### 第58轮增量

PR55已合并。R05-B将现有业务写入入口在schema3下接到关系/根路径：通用Store更新走根域事务，设置、异常状态及审计/失败异常追加定向writer允许schema3并在写前写后做关系完整性校验；根写回保留未变子树原文、未知键及大整数精度。通用入口改应用/检查仍拒绝。验证四路并发无丢失、审计故障回滚、损坏拒绝不调用回调及六项变异捕获。启动仍拒绝schema3。下一步应用/检查/发布HTTP入口接专用writer，再进入C/D。见test-report-v0.58.md及PR最终CI。

### 第59轮增量

PR56已合并。R05-B应用链路HTTP入口在schema3下接专用writer：Store层Try*探测，create/saveDraft/checkRelease/release优先走关系事务；v2路径不变；创建避免重复审计；草稿审计带client/request。验证完整创建→发布序列、通用Store仍拒改应用、并发CAS与故障回滚。R05-B写入适配收口。下一步C无损快照、D显式apply。见test-report-v0.59.md及PR最终CI。

### 第60轮增量

PR57已合并。R05-C实现schema3无损快照：`ExportMySQLSnapshot`/`ImportMySQLSnapshot`支持schema_version=3，封装state_root、关系表原文与审计payload；抽取`applyMySQLApplicationLayout`与`scanMySQLApplicationLayout`。验证导出→空库导入→再导出保真、损坏拒绝、启动仍拒schema3。v2桥接不替代。下一步R05-D显式apply与版本切换门禁。见test-report-v0.60.md及PR最终CI。

### 第61轮增量

PR58已合并基线。R05-D：`ApplyMySQLApplications`+CLI `--apply --expect-sha256`；`OpenMySQL`在合法schema3 handoff后启动。验证 plan/apply/导出、错 checksum、非法 version 拒绝及原 v3 回归。R05 主项待生产切换演练与多实例重启收口。见 test-report-v0.61.md 及 PR 最终 CI。

### 第62轮增量

PR59已合并基线。R05收口：`Apply`后重启保状态、双Store草稿CAS、编译态HTTP只读等价、双主机会话409。代码侧A–D闭环；下一步主队列 R06 授权拆表，R20承担正式多实例。见 test-report-v0.62.md 及 PR CI。

### 第63轮增量

PR60已合并基线。R06首轮：grants 目标表DDL、`PlanMySQLGrants`/`cognara-migrate-grants --check`、根域 grants 拆装与内部复制彩排。要求 schema3；目标 schema4 仅在计划中登记。下一步关系读写与显式 apply。见 test-report-v0.63.md 及 PR CI。

### 第64轮增量

PR61之后延续。R06 apply：`ApplyMySQLGrants`+CLI `--apply`；`OpenMySQL`/Store 在 schema4 从表重组 grants；根 writer 禁止经 body 改 grants。下一步 grants 关系 writer 与撤销幂等。见 test-report-v0.64.md 及 PR CI。

### 第65轮增量

R06 writer：`TryMutateGrantsV4` 与治理 grant HTTP；创建去重、撤销幂等与审计同事务。下一步 schema4 快照与 R20 多实例 grant 并发。见 test-report-v0.65.md 及 PR CI。

### 第66轮增量

R06 快照：schema4 export/import 与 grant 表原文保真。代码侧 R06 闭环；生产迁移与 R20 并发另计。见 test-report-v0.66.md 及 PR CI。

### 第67轮增量

R07 首轮：sessions/messages 目标表 DDL、`PlanMySQLSessions`/`cognara-migrate-sessions --check`、根域 sessions 拆装与内部复制彩排。见 test-report-v0.67.md 及 PR CI。

### 第68轮增量

R07 apply：`ApplyMySQLSessions`+CLI `--apply`；`OpenMySQL`/Store 在 schema5 从表重组 sessions；根 writer 禁止经 body 改 sessions。下一步 session 关系 writer 与 schema5 快照。见 test-report-v0.68.md 及 PR CI。

### 第69轮增量

R07 writer：`UpdateOperationalContext`/`TryMutateSessionsV5` 关系写；schema5 grant 读路径修复。下一步 schema5 快照与分页/脱敏专项。见 test-report-v0.69.md 及 PR CI。

### 第70轮增量

R07 快照：schema5 export/import 与会话/消息表原文保真。代码侧 R07 闭环；分页/脱敏/生产迁移与 R20 另计。见 test-report-v0.70.md 及 PR CI。

### 第71轮增量

### 第76轮增量

R08：schema7 快照 export/import 与 exceptions 表原文保真。代码侧 settings+exceptions 快照闭环；usage 域与 R20 另计。见 test-report-v0.76.md 及 PR CI。

### 第77轮增量

R08：usage plan + `cognara_fg_usage` DDL 与拆装彩排。下一步 usage apply+读（schema8）与 schema8 快照。见 test-report-v0.77.md 及 PR CI。

### 第78轮增量

R08：usage apply+读（schema8）与治理列表关系化。下一步 schema8 快照；R20 另计。见 test-report-v0.78.md 及 PR CI。

### 第79轮增量

R08：schema8 快照 export/import；settings+exceptions+usage 代码闭环。R20 / R07 分页脱敏 / 生产运维 apply 另计。见 test-report-v0.79.md 及 PR CI。

### 第80轮增量

R20：schema8 双 `OpenMySQL` 并发专项（草稿 CAS、usage 追加、异常状态）。下一步双 HTTP 主机与压力/SSE。见 test-report-v0.80.md 及 PR CI。

### 第81轮增量

R20：schema8 双 HTTP 主机会话重命名 CAS。R07：schema8 治理 usage 分页与撤权消息脱敏。修复 schema8 会话根写入与 grant writer。下一步 R20 压力/SSE/SLO。见 test-report-v0.81.md 及 PR CI。

### 第82轮增量

R20：schema8 双 HTTP 并发 grant 与 settings profile CAS；settings v7–8 关系化写修复。下一步压力/SSE/SLO。见 test-report-v0.82.md 及 PR CI。

### 第83轮增量

R20：schema8 双 HTTP continue-stream 完成重放（不重跑业务）与 /ready 并发探测。下一步正式压力、SSE 中断与 SLO。见 test-report-v0.83.md 及 PR CI。

### 第84轮增量

R20：schema8 治理 usage / 消息 load 只读并发；schema8 生产 apply 运维登记模板。下一步运维实填与正式压力/SLO。见 test-report-v0.84.md 及 PR CI。

### 第85轮增量

R20：agent-chat 取消/短 deadline、快照 import 彩排；R02：`r02-mysql-api-evidence-v0.85.md`。见 test-report-v0.85.md 及 PR CI。

### 第86轮增量

R20：治理 audit/exceptions 只读并发；schema8 exceptions 列表关系化修复；R02 证据 v0.86。见 test-report-v0.86.md 及 PR CI。

### 第87轮增量

R20：治理 grants 只读并发；双 HTTP 异常状态 CAS、应用草稿 CAS（PAGE-F2）；R02 证据 v0.87。见 test-report-v0.87.md 及 PR CI。

**建议下一轮（v0.88+）**：发布检查/发布 CAS 双 HTTP；治理 grants 变更并发（POST/DELETE）；正式压力/SLO 与运维登记表实填。

### 第88轮增量

R20：release-checks 陈旧修订、双主机发布 CAS、grant 并发撤权；R02 证据 v0.88。见 test-report-v0.88.md 及 PR CI。

**建议下一轮（v0.89+）**：模板创建应用双主机 CAS；治理 POST grant 与撤权交叉并发；正式压力/SLO 与运维登记表实填。

### 第89轮增量

R20：模板复制陈旧修订 CAS、双主机并发合法复制、grant 授撤交叉；R02 证据 v0.89。见 test-report-v0.89.md 及 PR CI。

**建议下一轮（v0.90+）**：会话创建/续聊双主机探测；正式压力/SLO；运维 runbook 登记表实填与慢查询证据。

### 第90轮增量

R20：peer 主机会话创建、会话列表并发读、跨主机 chat/load；治理 EXPLAIN 烟测与慢查询清单。见 test-report-v0.90.md 及 PR CI。

**建议下一轮（v0.91+）**：多会话连续写入专项（或 session writer 入口）；正式压力/SLO；runbook 登记表生产实填。

### 第91轮增量

R20/schema8：会话关系化 writer 挂接 HTTP、连续创建修复、`sessionsRelationalEqual`。见 test-report-v0.91.md 及 PR CI。

**建议下一轮（v0.92+）**：renameSession 等其余会话突变走 V5 writer；正式压力/SLO；runbook 生产登记表实填。

### 第92轮增量

R20：会话 rename/agent-chat 走 V5 writer；双轮 agent-chat 关系化验收。见 test-report-v0.92.md 及 PR CI。

**建议下一轮（v0.93+）**：轻量并发 agent-chat 只读重放（continue-stream）；正式压力/SLO；runbook 生产登记表实填。

### 第93轮增量

R20：`continue-stream` 32 路并发重放；SLO 探针基线表。见 test-report-v0.93.md 及 PR CI。

**建议下一轮（v0.94+）**：治理/消息 load 更高并发阶梯（记录耗时）；R21 正式 SLO 定标；runbook 生产登记表团队实填。

### 第94轮增量

R20：治理 `usage` 与消息 `load` 32→64 读阶梯（墙钟登记）；R21 SLO 定标草案；runbook 生产实填表。见 test-report-v0.94.md 及 PR CI。

**建议下一轮（v0.95+）**：多路径混合只读烟雾（单测编排）；R21 签核后同步告警；生产 runbook 首条实填验收。

### 第95轮增量

R20：混合只读烟雾单测编排；R21 告警 handoff；runbook 首条实填清单。见 test-report-v0.95.md 及 PR CI。

**建议下一轮（v0.96+）**：settings/public 双主机读并发；SSE 断连恢复探测；runbook 首条生产实填（团队）。

### 第96轮增量

R20/R01：governance 公共列表与 settings/public 双主机读并发；SSE 断连后重放。见 test-report-v0.96.md 及 PR CI。

**建议下一轮（v0.97+）**：治理 settings 写路径双主机 probe；agent-chat 进行中 turn 断连语义；runbook 首条生产实填（团队）。

### 第97轮增量

R20/R01/R14探针：settings/public 双主机写 CAS；进行中 turn 不可重放与断连语义文档。见 test-report-v0.97.md 及 PR CI。

**建议下一轮（v0.98+）**：general/profile settings 双主机写矩阵补全；混合烟雾纳入 public 写后读；runbook 首条生产实填（团队）。

### 第98轮增量

R20/R01：profile/general 双主机写 CAS；public 写后混合烟雾。见 test-report-v0.98.md 及 PR CI。

**建议下一轮（v0.99+）**：settings 写矩阵纳入混合烟雾写阶段；R21 签核试点填表；runbook 首条生产实填（团队）。

### 第99轮增量

R20/R01/R21：混合烟雾写+读编排；SLO 签核试点表。见 test-report-v0.99.md 及 PR CI。

**建议下一轮（v1.00+）**：完整写读烟雾纳入 runbook 默认命令；R21 正式签核；生产 runbook 首条实填（团队）。

### 第100轮增量（v1.00）

R20/R21：runbook 默认烟雾单测门与命令文档；R21 正式签核记录表。见 test-report-v1.00.md 及 PR CI。

**建议下一轮（v1.01+）**：默认门纳入 CI 可选 job；签核记录团队实签；生产 runbook 首条实填验收。

### 第101轮增量（v1.01）

R20/R21：GitHub Actions 默认烟雾 workflow；首条实填与签核指引。见 test-report-v1.01.md 及 PR CI。

**建议下一轮（v1.02+）**：validation 内拆 runbook job 摘要 artifact；团队完成 signoff-record 首条签字；生产 checklist 首条勾选。

### 第102轮增量（v1.02）

R20/R21：validation 与 runbook workflow 上传烟雾摘要 artifact；进度补充 `progress-snapshot-2026-11-18-r20.md`。见 test-report-v1.02.md 及 PR CI。

**建议下一轮（v1.03+）**：签核记录首条工程+运维签字；生产 checklist 首条实填；R02 进度快照与 R18 抽页回归。

### 第103轮增量（v1.03）

R18/R02/R21：浏览器抽页回归；R02 进度快照；工程彩排台账与 checklist 彩排子表。见 test-report-v1.03.md 及 PR CI。

**建议下一轮（v1.04+）**：台账两方彩排签字；生产 §3 首条实填；R18 截图基线或扩大抽页集合。

### 第104轮增量（v1.04）

R18/R02/R21：抽页扩展 + CI 截图 artifact；彩排 §3 模板与台账确认最小集。见 test-report-v1.04.md 及 PR CI。

**建议下一轮（v1.05+）**：台账实签；生产 §3 首条实填；R18 像素 diff 或 Playwright 快照门。

### 第105轮增量（v1.05）

R18：提交 PNG 基线 + CI 像素对比门；R21：团队签核提请单。见 test-report-v1.05.md 及 PR CI。

**建议下一轮（v1.06+）**：台账与 SLO 实签；生产 §3 首条实填入账；R18 扩大抽页或降低 flake 阈值评审。

### 第106轮增量（v1.06）

R18 扩至 10 抽页；彩排 JSON 校验与签核就绪门。见 test-report-v1.06.md 及 PR CI。

**建议下一轮（v1.07+）**：团队实签与生产 §3 入账；R18 聊天/编辑抽页；正式压测或 R21 告警 dry-run。

### 第107轮增量（v1.07）

R18 聊天/编辑抽页；R21 告警 dry-run JSON 与校验。见 test-report-v1.07.md 及 PR CI。

**建议下一轮（v1.08+）**：团队实签与生产 §3 真仓入账；R18 发布审阅抽页；压测 harness 或告警规则首次导入 staging。

### 第108轮增量（v1.08）

R18 发布审阅抽页；R21 staging 告警 Prom 渲染 artifact；生产 §3 形状样例。见 test-report-v1.08.md 及 PR CI。

**建议下一轮（v1.09+）**：生产 §3 真值入账与 SLO 实签；staging 告警人工导入验收；R18 应用详情/会话列表抽页。

### 第109轮增量（v1.09）

R18 应用详情与会话侧栏抽页；staging 告警渲染校验与导入验收清单。见 test-report-v1.09.md 及 PR CI。

**建议下一轮（v1.10+）**：生产 §3/SLO 实签入账；staging 告警 import 工单闭环；R18 发布历史/用量页抽页。

### 第110轮增量（v1.10）

R18 发布历史与治理用量抽页；staging import 工单 JSON 校验。见 test-report-v1.10.md 及 PR CI。

**建议下一轮（v1.11+）**：工单 CLOSED 实填；生产 §3/SLO 实签；R18 异常/审计 Tab 抽页或压测 harness 起步。

### 第111轮增量（v1.11）

R18 运营异常/审计 Tab；staging CLOSED 形状样例；SLO 待签 JSON；R09 ready probe harness。见 test-report-v1.11.md 及 PR CI。

**建议下一轮（v1.12+）**：团队真 CLOSED/SIGNED 入账；message load SLO 阶梯扩面；R18  grants Tab 或治理窄屏第二组。

### 第112轮增量（v1.12）

message load **96w** 阶梯 harness；治理窄屏 R18；SLO SIGNED 形状 JSON。见 test-report-v1.12.md 及 PR CI。

**建议下一轮（v1.13+）**：团队真实 SIGNED/CLOSED；usage 96w 阶梯；R18 服务/资产窄屏或 SLO 门槛写入告警 dry-run。

### 第113轮增量（v1.13）

usage 96w；告警 dry-run 6 规则（含 tier96）；服务/资产窄屏 R18。见 test-report-v1.13.md 及 PR CI。

**建议下一轮（v1.14+）**：团队真实台账入账；Prom 规则与监控指标接线草案；R18 build 窄屏或 chat 窄屏。

### 第114轮增量（v1.14）

指标接线草案；生产台账 PENDING 模板；build/chat 窄屏 R18。见 test-report-v1.14.md 及 PR CI。

**建议下一轮（v1.15+）**：台账 ENTERED 实填；exporter 骨架或 recording rule 样例文件；R18 设置/租户窄屏第二组。

### 第115轮增量（v1.15）

ENTERED 台账形状；recording rule YAML 样例；设置窄屏 R18。见 test-report-v1.15.md 及 PR CI。

**建议下一轮（v1.16+）**：团队真 ENTERED；exporter OpenMetrics 样例 endpoint 文档；R18 登录/禁止页窄屏或 CI 摘要 tier 日志 artifact。

### 第116轮增量（v1.16）

R09 tier JSON artifact；OpenMetrics exporter 样例；登录/禁止窄屏 R18。见 test-report-v1.16.md 及 PR CI。

**建议下一轮（v1.17+）**：真 ENTERED 与 exporter 上线；tier 摘要入账 runbook；R18 编辑器/发布窄屏或 mixed-read 告警绑定。

### 第117轮增量（v1.17）

tier 摘要校验与彩排打包 JSON；编辑器/发布窄屏 R18。见 test-report-v1.17.md 及 PR CI。

**建议下一轮（v1.18+）**：bundle CONFIRMED 实填；最小 exporter 只读端点 mock；R18 app-detail / session 窄屏。

### 第118轮增量（v1.18）

CONFIRMED 彩排形状；OpenMetrics loopback mock；detail/session 窄屏 R18。见 test-report-v1.18.md 及 PR CI。

**建议下一轮（v1.19+）**：团队真 CONFIRMED/ENTERED；mock 与 tier 摘要数值联动；R18 build-releases 窄屏或 governance-usage 窄屏。

### 第119轮增量（v1.19）

tier→OpenMetrics 渲染与校验；build-releases/governance-usage 窄屏。见 test-report-v1.19.md 及 PR CI。

**建议下一轮（v1.20+）**：真 CONFIRMED/ENTERED；runbook 读阶段写入 mock；R18 governance-audit/ops 窄屏补齐。

### 第120轮增量（v1.20）

runbook read→OpenMetrics CI snapshot；治理 audit/ops 窄屏 R18。见 test-report-v1.20.md 及 PR CI。

**建议下一轮（v1.21+）**：mixed-read 写入 snapshot；团队真台账；R18 全站窄屏覆盖率盘点或阈值收紧。

### 第121轮增量（v1.21）

mixed-read→OpenMetrics；窄屏覆盖率 JSON；bundle 含 snapshot artifact。见 test-report-v1.21.md 及 PR CI。

**建议下一轮（v1.22+）**：snapshot 全 6 指标 CI 联动验收；团队真 CONFIRMED；R18 阈值审计收紧（删冗余容差）。

### 第122轮增量（v1.22）

OpenMetrics CI snapshot 全 6 指标联动验收；R18 像素容差审计 artifact；bundle `openmetrics_linked_metric_count`。见 test-report-v1.22.md 及 PR CI。

**建议下一轮（v1.23+）**：在绿 CI 下逐步下调窄屏容差；团队真 CONFIRMED/ENTERED；生产 exporter 只读上线草案评审。

### 第123轮增量（v1.23）

治理窄屏容差收紧；生产 exporter 只读评审清单。见 test-report-v1.23.md 及 PR CI。

**建议下一轮（v1.24+）**：第二轮容差收紧（login/assets 等）；评审清单签字列实填；R18 manifest 与 spot 数自动对账。

### 第124轮增量（v1.24）

第二轮窄屏容差；exporter 评审签字形状；R18 manifest↔smoke↔PNG 对账（38）。见 test-report-v1.24.md 及 PR CI。

**建议下一轮（v1.25+）**：团队 APPROVED/真 CONFIRMED；CI 后第三轮容差；manifest 对账进 release 门禁文档。

### 第125轮增量（v1.25）

第三轮窄屏容差；APPROVED 形状 JSON；release 门禁 R18 manifest。见 test-report-v1.25.md 及 PR CI。

**建议下一轮（v1.26+）**：绿 CI 后第四轮容差（chat/editor）；团队真 APPROVED；release 门禁写入 runbook §4 勾选项。

### 第126轮增量（v1.26）

第四轮 chat/editor 窄屏容差；runbook §4 G1–G4 发版勾选与彩排 C4。见 test-report-v1.26.md 及 PR CI。

**建议下一轮（v1.27+）**：第五轮容差（session/publish 窄屏）；团队真 APPROVED/ENTERED；G4 与生产 §3 实填联动验收。

### 第127轮增量（v1.27）

第五轮 session/publish 窄屏容差；§3↔G4 联动文档与清单 #7/C5。见 test-report-v1.27.md 及 PR CI。

**建议下一轮（v1.28+）**：第六轮容差（app-detail/forbidden 窄屏）；台账 ENTERED 真值；G4 artifact SHA256 写入 exporter 评审附录。

### 第128轮增量（v1.28）

第六轮 app-detail/forbidden 窄屏；ENTERED 台账 G4 SHA256 字段；exporter 附录 A。见 test-report-v1.28.md 及 PR CI。

**建议下一轮（v1.29+）**：第七轮容差（build-apps/services 窄屏）；团队真 ENTERED/exporter APPROVED；staging 告警工单形状 CLOSED 样例（非真关闭）。

### 第129轮增量（v1.29）

第七轮 build-apps 窄屏（services 保持）；staging **`CLOSED_SHAPE_ONLY`**；真 CLOSED 拒 PLACEHOLDER。见 test-report-v1.29.md 及 PR CI。

**建议下一轮（v1.30+）**：第八轮容差（assets/settings 窄屏）；工单真 CLOSED 与 ENTERED/APPROVED 真签；bundle 挂工单号形状字段。

### 第130轮增量（v1.30）

第八轮 settings 窄屏（assets 保持）；bundle `staging_import_ticket_ref`；`CONFIRMED_SHAPE_ONLY`。见 test-report-v1.30.md 及 PR CI。

**建议下一轮（v1.31+）**：第九轮容差（governance-ops 窄屏）；bundle 真 CONFIRMED 与台账 ENTERED 同工单号校验脚本。

### 第131轮增量（v1.31）

第九轮 governance-ops 窄屏；`apply_change_ticket_ref` 对账脚本。见 test-report-v1.31.md 及 PR CI。

**建议下一轮（v1.32+）**：R18 容差 v1.31 收口盘点 artifact；三证同工单（bundle/ledger/staging）可选强化；真 ENTERED 与 exporter APPROVED 联席形状。

### 第132轮增量（v1.32）

容差 rollup artifact；三证工单校验；ENTERED↔exporter 联席形状。见 test-report-v1.32.md 及 PR CI。

**建议下一轮（v1.33+）**：第十轮容差（仅新 slug 或 JSON 基线刷新）；真签三联验收 runbook 实填挂钩。

### 第133轮增量（v1.33）

`governance-grants-narrow` 显式容差；三联 runbook 门 C6/#8。见 test-report-v1.33.md 及 PR CI。

**建议下一轮（v1.34+）**：services 窄屏显式比率（待定 CI）；rollup 分段归档策略；真签 bundle 写入变更单号 shape 交叉表。

### 第134轮增量（v1.34）

services-narrow 锁定清单；rollup 分段；变更单交叉表形状。见 test-report-v1.34.md 及 PR CI。

**建议下一轮（v1.35+）**：apps-narrow 等新 slug 编目；真签交叉表入库工单模板；R18 全 elevated spot 与 CI 差值对照 artifact。

### 第135轮增量（v1.35）

apps / ops-exceptions-narrow 编目；像素门 measured + headroom artifact；交叉表入库模板。见 test-report-v1.35.md 及 PR CI。

### 第136轮增量（v1.36）

batch-1 implicit-default 编目；headroom 阈值门；交叉表 `FILLED_SHAPE_ONLY`。见 test-report-v1.36.md 及 PR CI。

### 第137轮增量（v1.37）

batch-2 编目完成；headroom↔SLO 联动；`WAREHOUSE_TICKET_SHAPE_ONLY`。见 test-report-v1.37.md 及 PR CI。

### 第138轮增量（v1.38）

收紧候选 artifact；headroom 团队 enforce 开关；crossref-bundle 联席。见 test-report-v1.38.md 及 PR CI。

### 第139轮增量（v1.39）

publish-review headroom；services-narrow v1.34 推迟说明；enforce workflow；CONFIRMED 联席。见 test-report-v1.39.md 及 PR CI。

### 第140轮增量（v1.40）

chat-user 容差；headroom 子集 enforce；签字名形状 bundle。见 test-report-v1.40.md 及 PR CI。

### 第141轮增量（v1.41）

build-apps-narrow defer；subset + CONFIRMED bundle（无新容差 ratio）。见 test-report-v1.41.md 及 PR CI。

**建议下一轮（v1.42+）**：publish-review v1.39 hold 下的 defer 文档；子集含 elevated narrow；ledger ENTERED 与 CONFIRMED bundle 工单同桌验收。
