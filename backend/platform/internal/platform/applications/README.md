# applications

F1/F2/F3：应用与模板配置、发布快照、当前发布指针、应用会话绑定和Turn引用。

当前仅目录占位与职责说明，尚无业务实现。

### MySQL审计迁移预检查

```sh
# COGNARA_MYSQL_DSN由部署环境注入。先验证副本和备份，不在命令行写凭据。
cognara-migrate-audit --check
cognara-migrate-audit --apply
cognara-migrate-audit --check
```

检查报告为JSON，status为READY（v1可迁移）或ALREADY_APPLIED（v2审计验证通过），附源/目标版本和事件数。失败非零退出。--check不建表/播种/修改，--apply仍重新校验，不因上一次检查成功跳过检查。顶层Audit大小写别名需先显式处理，不会自动丢弃源数据。完整边界见docs/development/test-report-v0.28.md。
