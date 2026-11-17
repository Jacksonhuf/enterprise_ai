# F/G MySQL关系模型与迁移事务设计

版本1.3；2026-11-17；基线PR47。R04设计交付；R05已实现只读预检、内部DDL/原文重组、锁源复制与根交接演练，具体增量见第8–10节。依据：现有State/App/Release/Check及HTTP实现、MySQLschema1/2和既有API契约。设计不改变一个Go平台服务的边界，前端/API响应暂不变。

**实施状态：运行时仍使用schema1/2，尚未开启schema3。** 公共命令仍只提供应用迁移预检，不创建目标表、不复制数据、不切换读写。内部DDL/结构校验器只在隔离测试库执行，供后续apply复用；当前没有公共建表或迁移入口。以下为后续迁移约束；不能把设计或PLAN_VALIDATED当作迁移完成。

## 1. 存储归属与版本

| 阶段 | 权威数据 | 版本/兼容策略 |
|---|---|---|
| 当前schema1 | 聚合状态，包括审计 | 已有显式审计迁移到schema2；应用预检拒绝schema1 |
| 当前schema2 | 聚合业务状态 + 独立审计表 | 继续正常运行；本轮预检只读这个版本 |
| R05目标schema3（保留版本） | 应用头/草稿/发布/检查关系表 + 原审计表；其余业务暂留聚合 | 必须先完成writer、reader、导入导出、健康检查及旧投影适配，再显式切换；当前运行时拒绝3 |
| R06–R08后续版本 | 授权、会话/消息/Turn、治理数据逐批迁移 | 每批独立版本和恢复/兼容门禁；不复用schema3偷偷扩大语义 |

保留State单例行作为**迁移屏障**，直到剩余聚合域迁出；移除应用JSON本身不等于移除全局写锁。新关系表的定向事务和多实例并发属于R05后续验收，不能由本轮预检宣称已实现。

## 2. 公共约束

- 所有业务读写必须带可信tenant_id；租户/对象复合主键和外键不允许跨租户引用。身份、数据范围仍由G1判定，外键不替代授权。
- tenant_id使用VARBINARY(64)，现有生成的对象/用户/版本标识使用VARBINARY(512)，按字节精确区分大小写。应用当前在State中按app_id全局索引，schema3兼容期保留UNIQUE(app_id)，检查ID同理；不趁迁移引入当前域模型无法表达的同名跨租户对象。
- JSON配置、快照与扩展保留原始JSON值。未知顶层扩展和子对象扩展不得被typed marshal静默丢弃；未来迁移需要raw-envelope拆装。投影字段大小写别名或不一致先阻断，不擅自选值。
- 时间原值用VARCHAR(64)保留RFC3339Nano及偏移；需要时间查询时另加UTC投影。不要把纳秒强制截为DATETIME(6)后宣称原值无损。排序/比较沿用现有API逻辑，不直接改为SQL默认排序。
- 版本沿用不可重用的revision字符串；CAS冲突返回既有409。历史发布不可变；历史恢复只创建新草稿，不切换线上指针。
- API层现有403/404存在性保护、分页游标和权限脱敏规则保持；关系表不能绕过它们。

## 3. R05四张表（DDL已实现，当前仅隔离测试执行）

| 表 | 列与约束 | 索引/引用 |
|---|---|---|
| cognara_fg_applications | tenant_id VARBINARY(64)、app_id VARBINARY(512)、agent_id/owner_id VARBINARY(512)、current_release_id VARBINARY(512) NULL、updated_at VARCHAR(64)、metadata JSON、releases_is_null BOOLEAN；其余非NULL | PK(tenant_id,app_id)；UNIQUE(app_id)兼容现有State；INDEX(tenant_id,owner_id,app_id) |
| cognara_fg_application_drafts | tenant_id、app_id、revision VARBINARY(512)、config JSON；均非NULL | PK(tenant_id,app_id)；FK(tenant_id,app_id)→applications；revision用于CAS |
| cognara_fg_application_releases | tenant_id、app_id、release_id VARBINARY(512)、ordinal BIGINT UNSIGNED、snapshot JSON、published_at VARCHAR(64)、published_by VARBINARY(512)、metadata JSON；均非NULL | PK(tenant_id,app_id,release_id)；UNIQUE(tenant_id,app_id,ordinal)；FK→applications；保留State发布数组顺序 |
| cognara_fg_application_checks | tenant_id、check_id VARBINARY(512)、app_id、draft_revision、user_id VARBINARY(512)、expires_at VARCHAR(64)、validation_id VARBINARY(512)、metadata JSON；均非NULL，validation_id可为空字符串以兼容历史 | PK(tenant_id,check_id)；UNIQUE(check_id)；INDEX(tenant_id,app_id,check_id)；FK→applications |

所有表InnoDB。VARCHAR文本显式utf8mb4；业务标识不依赖文本排序规则。metadata保留业务字段之外的原始扩展；draft/config和release/snapshot分开持有原始配置。releases_is_null用于区分历史null与空数组，保证重组语义。草稿只允许一个；每个应用必须有草稿。

current_release_id在子表完成装载后建立复合外键(tenant_id,app_id,current_release_id)→releases。父表显式建立INDEX(tenant_id,app_id,current_release_id)以支撑该引用。装载时先以NULL插入应用头，再写草稿/发布/检查，最后设置线上指针；不能要求MySQL延迟检查外键。没有已发布版时NULL合法；非NULL必须在本应用发布集中。发布快照禁止UPDATE/DELETE，应用删除/归档需求未冻结，不在迁移中引入级联删除。

本轮预检验证标识长度、对象引用、发布重复/顺序可表达性、时间格式和检查归属。过期或旧revision的检查仍是合法历史记录，分别计数，不把它们视为当前可发布凭证，不删除它们。预检不重新评估模型能力或A–E引用；这些仍由发布检查负责。

## 4. R06–R08及可靠运行目标表

| 域/目标表 | 主键与必要索引 | 原子性与兼容要求 |
|---|---|---|
| grants | (tenant_id,grant_id)；主体+资源+action+scope检索；有效授权元组唯一约束需表达撤销历史 | 创建去重、撤销幂等；与审计/outbox同事务；管理权不等于数据权 |
| sessions | (tenant_id,session_id)；app_id+owner_id+排序键；兼容全局session_id | 标题CAS、用户/应用隔离、分页不漏不重；撤权历史仍脱敏 |
| messages | (tenant_id,session_id,message_id)；会话顺序号唯一 | 内容/证据引用/完成标记同事务；保留原始result及脱敏标记；不分裂已完成答案 |
| turns | (tenant_id,turn_id)；session+状态+更新时间 | 持久状态转换CAS；中断不是成功，完成结果重放不等于运行中恢复 |
| exceptions | (tenant_id,exception_id)；request_id、状态/时间索引 | 确认/解决/重开CAS；状态处理不重试业务执行 |
| usage | (tenant_id,event_id)；能力/应用/发生时间索引 | 幂等去重；缺失词元明确未知，不冒充账单 |
| settings | (tenant_id,owner_id,section)；revision | profile/general归用户，public归企业；保留旧键迁移；CAS/空值和无操作审计语义不变 |
| receipts | (tenant_id,producer_id,event_id) | 明确本地接收和提供方送达回执；重投不重复业务/用量 |
| outbox | (tenant_id,event_id)；状态+next_attempt_at索引 | 业务提交与待发记录同事务；有限重试/退避/失败可查；投递失败不伪报业务回滚 |
| audit（现有） | (tenant_id,event_id)；当前sequence全局唯一 | 保留现有不可变事件/去重与顺序；全局序号锁仍是未来并发优化点，不先改契约 |

上述除audit之外尚未启用关系表。外部A–E结果、用户组身份源和商业模型仍通过现有适配器接入，不搬进F/G数据库冒充权威源。

## 5. 事务与锁顺序

| 操作 | 同事务工作 | 失败/重试语义 |
|---|---|---|
| 创建应用 | 应用头、草稿、审计/outbox；模板复制绑定所选源revision | 新草稿不得继承源会话/授权/发布；未知提交结果不得盲重试 |
| 保存/恢复草稿 | 锁应用头→草稿CAS→更新revision/时间→审计/outbox | 旧revision409；拒绝ABA；恢复不修改发布集合/指针 |
| 创建检查 | 外部验证在事务外；事务内重新核对应用/revision后插入检查 | 草稿变化则检查不能成为新草稿的发布凭证 |
| 发布 | 锁应用头→草稿→检查；核对用户/revision/当前指针/有效期；插入不可变快照→换指针→消费检查→审计/outbox | 任一步失败整体回滚；同一检查只能消费一次；HTTP超时后先查询结果，不自动重复发布 |
| 迁移 | 锁迁移屏障→重新检查版本/原始数据/目标空表→批量复制→重组对账→删除apps/checks聚合键→schema版本更新 | DDL独立提交；数据与版本在一个事务提交；不把DDL残留说成事务已回滚 |

正常单应用操作按tenant_id/app_id固定顺序取得锁；涉及多个应用的模板复制按键排序，先捕获源快照，再在创建事务核对源版本。迁移期旧writer必须停止，不能让schema2进程在切换后继续用原投影写入；schema3 writer不得回退写聚合副本。禁止双权威数据。

## 6. 迁移、对账与恢复步骤

1. 基线：保存已验证的完整schema2快照、原始状态SHA256、审计计数和数据库备份，验证恢复到隔离库。快照本身不等于生产备份体系。
2. 本轮命令：`cognara-migrate-applications --check`，DSN只从环境变量读取。输出PLAN_VALIDATED、source/target版本、计数、源状态SHA256、采样时点，read_only=true、apply_supported=false。schema1、未知版本、损坏引用、别名歧义、已有任何保留目标表都拒绝。不创建空库/表，不seed，不FOR UPDATE。
3. 后续apply先执行DDL并校验列、主键、索引、外键、字符集和InnoDB；目标表有数据/不兼容时阻断，不能擅自覆盖。预检通过不是apply令牌：数据库在预检后可能变化。
4. 停止旧writer；在迁移屏障下重新读取并验证。分别复制应用头/草稿/发布/检查，保留未知JSON扩展；对账应用、租户、检查、发布数量及顺序、所有版本/指针、配置值、null/[]形态和扩展语义。
5. 同一事务中移除已迁出的apps/checks键并提升至schema3；**不能保留聚合副本作为失败时的隐式fallback**。提交未知则检查数据库实际版本及对账，不盲重跑。其余聚合域及未知扩展原样保留。
6. 启动支持schema3的新reader/writer。真实MySQL运行应用创建→CAS→检查→并发发布→会话→证据→导出→恢复完整回归；所有原投影/Check/export/import必须支持3或显式拒绝，不能返回默认成功。
7. 回退：切换前失败保持schema2，DDL可能残留需显式检查；切换后禁止直接降低版本号。无新写入可从已验隔离备份恢复并核对；已有新写入须先导出新模型并完成反向转换/恢复验证，否则只允许前向修复。RPO/RTO由生产验收定义并实测。

## 7. 运行与验收

```bash
go build -C backend/platform -tags=nomsgpack -o /tmp/cognara-migrate-applications ./cmd/cognara-migrate-applications
# 预先设置COGNARA_MYSQL_DSN；不要把凭据写入命令历史或文档。
/tmp/cognara-migrate-applications --check
```

可用`--mysql-dsn-env`指定已有环境变量名。命令没有`--apply`，缺少`--check`/附加参数拒绝执行；失败不输出成功JSON，错误不包含DSN或业务内容。报告仅计数和校验和，不返回配置、应用名称或用户标识。SHA256针对本次聚合状态原始字节，不覆盖独立审计表全部内容，不是全库备份校验和。

本轮验收：纯函数损坏/引用/租户/历史检查回归、CLI参数/脱敏、真实MySQLschema2只读/目标表冲突/未知版本/取消/写锁并存、编译后二进制。正式切换、表结构校验器、重组对账、schema3导入导出、多实例CAS与生产备份恢复**仍是R05及后续任务**。

## 8. 第45轮实现状态

- `migrations/002_applications.sql`及内部建表函数：四张InnoDB表、显式索引和复合租户外键；先建父/子表，再添加当前发布外键。没有IF NOT EXISTS覆盖既有目标，首次失败停止，DDL残留不自动删除。
- 结构校验器支持DB和同一事务：列类型/可空/默认值/生成属性、表及文本列排序规则、完整索引顺序/唯一性/前缀/可见性、外键列/引用库/级联规则、CHECK语义和ENFORCED状态。发布不可变仍需未来writer及权限约束；这套DDL本身不禁止对非当前发布行的任意UPDATE。
- 原始记录拆装器分离应用头、草稿、发布数组项及检查，未知根/头/配置/发布/检查扩展均保留，记录关联投影必须与原文匹配。保持发布顺序和null/[]、checks缺失/null/对象形态。未来SQL落表还须保存这些形态元数据并检查所有列投影，不能丢弃内存布局中的shape标记。
- 对账使用json.Number，避免float64吞掉大整数/高精度数值；允许JSON对象键顺序、空白、等价字符串转义变化，不声称字节相同。原始状态SHA256仍针对读取的源字节。
- `--check`已真实调用拆分→重组→对账，成功报告新增`round_trip_verified:true`；`apply_supported:false`不变，仍拒绝已存在的目标表。没有启用schema3 reader/writer、写入影子副本或修改源库。

下一批R05是事务内SQL复制、原文/投影验证与数据库重载对账，再适配schema3读写/投影/快照导入导出，全部通过前不提升schema_version。

## 9. 第47轮：原文持久化与独立数据库重组

四张尚未公开启用的目标表各新增`raw_payload LONGBLOB NOT NULL`：分别保存应用头、草稿配置、发布完整记录、检查完整记录。原有`metadata/config/snapshot JSON`列降为原文的数据库查询投影；不可用MySQL归一化后的JSON列反向生成权威原文。标识、时间、版本、指针、发布顺序等标量投影仍逐字段核对。原始JSON的数值表示、高精度小数和扩展字段由二进制列保留。

新增内部`cognara_fg_application_envelope`：`id TINYINT UNSIGNED PRIMARY KEY CHECK(id=1)`、`root_payload LONGBLOB NOT NULL`、`checks_kind VARBINARY(8) NOT NULL`。它保存本次隔离复制的剩余根对象及checks缺失/null/object形态；应用的releases null/[]沿用父表标记。该表不设租户访问API，根对象本身是原聚合状态的跨租户迁移封套，不能作为普通租户查询入口。公共只读预检将它列为保留目标，已有旧版内部DDL也不自动ALTER接管。

这是隔离迁移验证的持久化布局，尚非运行时schema3的生产写入路径。当前源State仍是唯一运行时权威，内部复制函数仅供回滚演练和隔离测试。后续接schema3时，剩余聚合域必须继续由单一已选定的根存储持有；不得同时让state.body与此envelope根副本接受业务写入。应在迁移屏障事务内决定根封套归属并改造原投影，不能直接把本次隔离副本当作实时影子库。

数据库重载不接受源JSON或内存layout：在调用方同一一致性事务中读取封套、四表原文和发布ordinal，校验连续顺序、父子归属、记录数、原文/SQL投影一致性后重组。复制阶段再与源JSON严格对账，允许键顺序、空白、字符串转义差异，数值字面值不归一化。完整原始字节SHA不保证相同。提交/关闭连接池/重新连接后的隔离测试检验独立持久化；这不是正式运行时重启或生产备份恢复验收。

## 10. 第50轮：根聚合单权威选择与交接演练

选定正式schema3后剩余聚合域继续由`cognara_fg_state.body`持有，State行继续作为迁移屏障；应用封套只持有checks形态，root_payload必须为空对象哨兵，不再接受第二份根业务数据。内部交接演练先在State锁下复制对账，再从body移除apps/checks、清空封套根，使用State根与关系原文重组并对原源严格核对。后续根更新直接从State读取，遇到封套非空立即拒绝，不隐式选择旧副本。

目前仅内部回滚事务内演练，schema_version仍为2。这个中间状态不能提交或暴露给schema2运行时；公共命令没有激活入口。正式版本切换必须同时完成reader/writer、所有旧投影和快照恢复接线，才可将交接与版本更新原子提交。交接中的任何错误均由演练外层回滚源与目标。
