# MySQL 迁移摘要与跨平台升级

旧程序对 Go 内嵌 SQL 的原始字节计算 SHA-256。Git 的 LF/CRLF checkout 差异会导致同一提交在不同平台生成不同迁移摘要。本轮对独立合成测试库的只读核对发现：19 条记录均对应 Windows 构建资源，其中 17 条与仓库原始 blob 的摘要不同；迁移 SQL 本身没有业务内容变更。

新校验保留原始字节 SHA 校验，并仅接受 SQL 字面量和引用标识符之外的 LF/CRLF 等价形式。字符串、引用标识符、MySQL 可执行注释和优化器提示的内容不变。SQL 类型、表名、默认值、空格或注释文字发生变化，仍报告 `migration checksum changed`；引号内的 LF 与 CRLF 可能代表不同数据，不能当作排版差异放行。当前迁移文件的两种 checkout 形式均有回归覆盖。

这项修改不编辑历史 SQL，不重新执行已登记迁移，不改写已有摘要和 `applied_at`，新写入仍使用构建资源的原始摘要。由此保留原有部署与其旧程序之间的摘要约定；新程序可以验证合法的历史 LF、CRLF 或逐条混合来源记录。回退仍应使用与原部署兼容的程序、schema、对象和 key，不能把修复后的读取兼容性理解为任意历史二进制都支持跨平台降级。

校验失败时保留原始数据库和证据，检查具体迁移文件与构建资源，不要删除或任意重写迁移账本来绕过错误。MySQL DDL 的隐式提交和分步恢复语义保持原样，本补丁不把整个迁移流程宣称为单事务。

`TestMigrationChecksumsOnlyIgnoreNonLiteralLineEndings` 覆盖真实内容、字面量/标识符换行、转义、双引号、普通注释、条件注释/提示、单独 CR、未闭合引号和注释等边界。`TestHistoricalMySQLMigrationNewlineForms` 对所有已内嵌 MySQL 文件交叉核对两种换行。`TestMySQLMigrationLineEndingsAndTampering` 使用真实新建 MySQL 库验证 LF、CRLF、逐条混合记录及原构建摘要均可升级；账本、时间、业务版本和通知去重键保持不变，未知或不同 SQL 的摘要仍失败且不被自动修复。真实库测试被双 Go 的 MySQL 结果审计列为必需。
