# schema8 R21 签核后告警同步清单（模板）

在 `mysql-schema8-r21-slo-draft.md` §3 签核后，由运维/SRE 将下列探针墙钟或错误率映射到监控系统。**签核前以下阈值留空，不开启生产告警。**

| 探针 / 测试 | 建议指标名 | 阈值（签核后填写） | 对应自动化 |
|-------------|------------|-------------------|------------|
| 双主机 `/ready` 48 workers | `fg.ready.smoke_wall_s` | < ___ s | `TestR20Schema8ParallelReadyAcrossDualHosts` |
| 混合只读烟雾 36 workers | `fg.read.smoke_wall_s` | < ___ s | `TestR20Schema8MixedReadOnlySmokeOrchestration` |
| `continue-stream` 32 路 | `fg.sse.replay_wall_s` | < ___ s | v0.93 |
| governance usage 64 阶梯 | `fg.gov.usage_tier64_wall_s` | < ___ s | v0.94 |
| message load 64 阶梯 | `fg.msg.load_tier64_wall_s` | < ___ s | v0.94 |

## 同步步骤

1. 从最新专项测试日志或 CI  artifact 抄录**验证环境**墙钟到上表草案。
2. 签核人对生产目标值签字后，乘以团队约定的安全系数（例如 ×1.5）写入监控。
3. 在变更单登记告警规则版本与 runbook §3「SLO 签核」行。
4. 首次生产烟雾：运行 `go test … -run TestR20Schema8MixedReadOnlySmokeOrchestration` 或等价手工探针，对照告警未误报。

## 签核

| 角色 | 日期（北京时间） | 监控规则 ID |
|------|------------------|-------------|
| SRE | | |