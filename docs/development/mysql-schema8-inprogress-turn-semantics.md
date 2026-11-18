# schema8 进行中 turn 与断连语义（探针说明）

与 R14「运行中断恢复」对齐的**当前实现边界**，非完整远端取消承诺。

| 场景 | 行为 | 自动化 |
|------|------|--------|
| 客户端在持久化前断连 / 取消 context | 不留下可重放消息 | `TestR20Schema8AgentChatCanceledContextBeforePersist` |
| 极短 deadline 的 agent-chat | 不持久化半截 turn | `TestR20Schema8AgentChatExpiresDuringUpstreamCall` |
| 库中存在 `Completed=false` 的 assistant 消息 | `continue-stream` → 409 `REFERENCE_NOT_READY`，双主机一致 | `TestR20Schema8DualHostContinueStreamInProgressTurn` |
| 断连后再完成一轮对话 | 仅已完成消息可双主机重放 | `TestR20Schema8InProgressTurnDisconnectThenCompletedReplay` |

**未包含**：进行中 SSE 流中途断线的增量续传、跨进程恢复未完成执行、远端取消回执。