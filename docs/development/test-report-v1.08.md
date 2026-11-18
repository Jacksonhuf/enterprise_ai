# 第108轮（v1.08）：R18 发布审阅抽页与 staging 告警渲染

## 变更摘要

- R18：**13** 抽页（+`publish-review` 抽屉）。
- `render-r21-alerting-staging-prometheus.py` → CI artifact `r21-staging-alert-render`。
- 生产 §3 JSON 形状样例（`schema8-production-log-shape-example.json`，待团队填）。

## 测试

```bash
python scripts/render-r21-alerting-staging-prometheus.py docs/development/fixtures/r21-alerting-dry-run.rules.json /tmp/out.prom.txt
python scripts/validate-rehearsal-log-record.py docs/development/fixtures/schema8-production-log-shape-example.json
# browser + MySQL per test-report-v1.07.md pattern
```