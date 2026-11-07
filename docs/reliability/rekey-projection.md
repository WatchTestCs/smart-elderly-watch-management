# PostgreSQL 位置投影密文轮换

`backend/cmd/rekey-projection` 处理 `suining_analytics.record.payload` 中一个明确 MySQL 来源的 `device_location` 坐标。分析消费者可能已经删除 MySQL outbox 事件，因此仅完成 [MySQL 字段与事件轮换](rekey-data.md) 不能证明 PG 不再依赖旧密钥。

## 配置与执行

先备份并恢复验证旧 key/lookupKey。所有相关读实例先部署 A+B reader，再将写实例切换为 active B，保持 lookupKey C 不变。该工具不改变服务配置，不自动停用 key，也不把 PG 变成 MySQL 故障时的业务主库。

明确配置三个私有文件：`MYSQL_DSN_FILE`、`PG_ANALYTICS_URL_FILE` 和 `DATA_ENCRYPTION_KEY_FILE`。新 keyring 必须含 active ID，与 `--previous-key-file` 的 lookupKey 相同，并能认证现有密文。命令从选定 MySQL 的 `analytics_source` 读取来源 ID，不能由命令行指定任意来源；主库或 PG 配置缺失时拒绝运行，不接受默认 PG 连接来代替明确配置。数据库角色按需授予读取源标识、读取 PG 投影及 apply 时更新 payload 的最小权限。

在后端模块目录构建 `go build -o rekey-projection ./cmd/rekey-projection` 后执行：

```text
rekey-projection --previous-key-file <私有旧配置文件> > pg-before.json
rekey-projection --previous-key-file <私有旧配置文件> --apply > pg-apply.json
rekey-projection --previous-key-file <私有旧配置文件> > pg-after.json
```

默认只读。`--batch-size` 为 1..1000，`--max-rows` 限制本来源行数，`--timeout` 设置整体时限，另有页面/字段字节限制和单次数据库操作 15 秒时限。命令最多使用两条 PG 连接。缺 key、错误 scope、损坏 JSON/密文、超大字段或预算耗尽的预检在首次写入前失败。

## 并发与恢复语义

只匹配所选 `source_id` 和精确的 `device_location` 实体名，按 bigint 主键游标分批处理。`lng`/`lat` 沿用原 `location` AEAD scope，其他载荷属性以原 JSON 表示保留。NULL 删除载荷不改，明文坐标仅计数，不自动迁移。两个坐标中任一个认证失败，整条载荷不更新。

更新必须同时匹配来源、实体、行 ID、旧 `last_event_id`、旧删除标志及 C collation 下的完整旧 JSONB 文本。较新事件即使携带相同载荷，也不能被旧批次覆盖；JSONB 数值相等但文本表示不同的并发修改同样会被保留。只更新 payload，不推进事件号、不修改 `synced_at`、不重建已删除行，不更新其他 MySQL 来源。

已经提交的行不会因后来失败而整体回滚。保留 A+B 后重跑：active 密文先认证再跳过，旧密文继续处理。发生并发冲突、认证错误、残留明文、超时或预算耗尽时 apply 返回非零，并保留不含行主键/原文/密文/连接凭据的聚合报告。来源只输出使用稳定 lookupKey 计算的 `source_ref`，不输出真实来源 UUID。

报告通过 `scope=postgresql_location_projection` 与 MySQL 报告区分，`encrypted_columns_clean` 仅表示本次选定来源和坐标字段扫描无残留，`retirement_authorized` 始终为 false。旧写入者、其他来源、历史明文健康 JSON、对象、binlog/副本、WAL、备份和回滚程序仍需独立盘点。没有写版本屏障和最旧备份恢复证据，不能删除 A。

## 测试

`TestProjectionRekeyPostgres` 在独立新建、迁移的真实 PG 库中验证只读、负数及数字 ID 分页、A/legacy→B、只用 B 的读取、NULL 删除、元数据/事件号/同步时间保持、其他来源/实体隔离、后序损坏阻止全部写入、明文及超大值、预算、取消，以及较新同载荷事件、删除和 JSONB 表示变化的 CAS 冲突。命令集成测试连接真实 MySQL+PG，验证来源确实取自 MySQL、其他来源保持原值，以及缺少明确配置时拒绝运行。

PG 库级测试和命令跨库测试分别进入双 Go 的 PG/MySQL 必需结果审计；缺失、失败或意外 skip 均不能被整体绿色状态掩盖。当前候选仍须以其提交 SHA 获取 CI 与独立审查，本工具不等于 P0-07 全部完成或真实生产密钥退役验收。
