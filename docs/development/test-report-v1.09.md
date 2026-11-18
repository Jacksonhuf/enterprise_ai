# 第109轮（v1.09）：R18 详情/会话抽页与 staging 告警渲染验收

## 变更摘要

- R18：**15** 抽页（+`app-detail-build`、`session-sidebar`）。
- `validate-r21-staging-alert-render.py`（CI + 人工导入前校验）。
- `mysql-schema8-r21-staging-import-acceptance.md`。

## 测试

```bash
python scripts/render-r21-alerting-staging-prometheus.py docs/development/fixtures/r21-alerting-dry-run.rules.json /tmp/out.prom.txt
python scripts/validate-r21-staging-alert-render.py /tmp/out.prom.txt
```