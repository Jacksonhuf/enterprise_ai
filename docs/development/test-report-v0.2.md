# 第二轮开发验证记录

日期：2026-11-15（中国标准时间）。范围：新增F/G模块、当前前端入口及选定的原引擎回归测试，不代表整个产品已验收。

| 检查 | 结果 | 验证内容 |
|---|---|---|
| Go新增模块 | 14个顶层测试通过 | 应用发布与版本绑定、权限隔离、服务幂等、模型工具调用及SQLite恢复等 |
| Go竞态检查 | 通过 | 核心模块使用-race运行，包括两个独立SQLite连接并发写入 |
| HTTP契约 | 54组请求/响应通过 | 实际A–E Mock与Go内部回调流量按现有v0.3契约验证 |
| 模型端到端 | 通过 | 浏览器侧API请求→原ReAct→本地模型协议夹具→A–E Mock→Go鉴权/审计→最终结果 |
| 原引擎回归 | 通过 | 空响应后恢复工具调用、模型请求会话标识、HTTP错误响应头保留 |
| 前端单元 | 3项通过 | SSE逐字节UTF-8/CRLF、拒绝访问错误、无效帧取消底层流 |
| 前端类型 | 通过 | 当前Cognara入口vue-tsc检查 |
| 前端构建 | 通过 | Vite生产构建；仍有大分包告警 |

## 可重复执行

```bash
make bootstrap
make check
COGNARA_CONTRACT_CAPTURE=/tmp/cognara-contract.jsonl go test -C backend/platform -tags=nomsgpack -p 1 -race -count=1 ./internal/platform/... ./cmd/cognara-dev
python scripts/validate-contract-capture.py /tmp/cognara-contract.jsonl
```

选定上游回归：

```bash
cd backend/platform
go test -tags=nomsgpack -p 1 ./internal/agent ./internal/models/api -run 'TestEmptyResponseRecoveryPreservesUserIntentAndTools|TestEndpointRequestIdentity|TestDoKeepsHeadersOnHTTPError'
```

`nomsgpack`只关闭本产品JSON接口不使用的Gin MessagePack编解码。初次并行执行竞态编译与前端打包时触发内存终止，改为低并发、顺序执行后通过；失败运行不计入通过结果。Node测试采用`node --import tsx --test`，避免CLI启动IPC管道在当前环境中的权限限制。

## 未覆盖的验收

商业模型服务实测（未提供凭据）、企业SSO、PostgreSQL与多节点部署、A–E真实服务联调、全量页面与Figma逐屏视觉验收均尚未完成。本地模型响应夹具只证明协议及集成路径，不证明任意商业模型的推理质量。
