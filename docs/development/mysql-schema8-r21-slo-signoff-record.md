# schema8 R21 SLO 正式签核记录（待团队签字生效）

试点数据见 [mysql-schema8-r21-slo-pilot-signoff.md](mysql-schema8-r21-slo-pilot-signoff.md)。**本表签字前不构成对外 SLO。**

## 1. 采纳目标（工程建议稿，待确认）

| 探针 | 建议正式门（验证环境×安全系数 1.5） | 试点观测 |
|------|--------------------------------------|----------|
| 默认写读烟雾 | 读阶段总轮 < **4 s** | ~2.2 s / 36w |
| 混合只读烟雾 | 总轮 < **3 s** | ~1.3 s / 36w |
| usage 阶梯 64w | 总轮 < **3 s** | ~1.4 s |
| message load 64w | 总轮 < **7 s** | ~4.6 s |

团队可改写上表后再签核；签核后同步 [mysql-schema8-r21-slo-draft.md](mysql-schema8-r21-slo-draft.md) §2 与 [mysql-schema8-r21-alerting-handoff.md](mysql-schema8-r21-alerting-handoff.md)。

## 2. 正式签核

| 角色 | 姓名 | 日期（北京时间） | 签字/工单号 |
|------|------|------------------|-------------|
| 平台工程负责人 | | | |
| 运维/SRE 负责人 | | | |
| 业务代表 | | | |

## 3. 生效登记

| 项 | 值 |
|----|-----|
| 监控规则版本 | |
| runbook §3 SLO 签核行 | |
| 首条生产实填 checklist | 见 `mysql-schema8-runbook-production-fill-checklist.md` |
| CI 默认门 Run URL（可选） | Actions：**F-G runbook default smoke** |