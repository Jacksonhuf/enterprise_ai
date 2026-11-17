# 第60轮：R05-C schema3无损快照导入导出

基线PR57。公共`ExportMySQLSnapshot`/`ImportMySQLSnapshot`在schema3下输出/接收`cognara-fg-snapshot`格式（schema_version=3）：`state_root`保存不含apps/checks的聚合根；`relational`保存应用封套与各关系表原文（header/draft/release/check LONGBLOB）；`audit`保存审计payload原文。导出前扫描关系表、验证投影并完整重组读取；导入时空库事务写入state(3)、`applyMySQLApplicationLayout`、审计，再重组校验。抽取`applyMySQLApplicationLayout`供复制演练与导入共用。

真实MySQL 8.4专项（本地8.4.9实例）：schema3 fixture导出→独立空库导入→再导出；state根扩展大整数保留，应用草稿原文逐字节一致；导入后关系读取器与`OpenMySQL`启动拒绝；损坏草稿投影后导出拒绝。v2桥接与数值归一化边界不变。未接显式apply/D激活，R05-C功能性交付、D仍未完成。完整CI见PR。