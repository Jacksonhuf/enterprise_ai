# 第110轮（v1.10）：R18 发布历史/用量抽页与 staging 工单模板

## 变更摘要

- R18：**17** 抽页（+`build-releases`、`governance-usage`）。
- `validate-staging-import-ticket.py` + 工单 JSON 模板。
- `mysql-schema8-r21-staging-import-ticket-closure.md`。

## 测试

```bash
python scripts/validate-staging-import-ticket.py docs/development/fixtures/staging-alert-import-ticket-example.json
# browser + MySQL: 同 test-report-v1.09.md
```