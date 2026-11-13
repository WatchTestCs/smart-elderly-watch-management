# MySQL、对象、密钥与 PG 投影的协调恢复回归

`TestImageS3TwoInstancesAndRealRestore` 使用真实 MySQL、PostgreSQL 与私有 S3 兼容服务，
不使用内存数据库、模拟对象服务或直接修改通知队列状态。CI 的 MySQL suite 强制该用例通过；
缺少连接而 skip 会被 `audit-go-results.py --mode mysql` 拒绝。

## 运行

只使用专用测试服务。MySQL 账号需要创建/删除测试数据库，PG 账号需要 CREATEDB；
S3 bucket 必须专用于合成测试数据。不可提供生产数据库、生产凭据或生产 bucket。
测试创建独立随机数据库，仅销毁自己创建的数据库和对象。准备与服务端兼容的
`mysql`、`mysqldump` 客户端并加入 PATH，然后在后端目录执行：

```sh
# 通过进程环境安全注入以下变量，不把口令写入日志：
# TEST_MYSQL_DSN, TEST_DATABASE_URL,
# TEST_S3_ENDPOINT, TEST_S3_BUCKET, TEST_S3_ACCESS_KEY, TEST_S3_SECRET_KEY
go test -json -race -count=1 -timeout 3m \
  -run '^TestImageS3TwoInstancesAndRealRestore$' ./internal/handlers > recovery.jsonl
```

须检查进程退出码，并确认该用例明确 pass，不能只看“命令已执行”。全量 CI 使用
`go test -json -race -count=1 -timeout 45m ./...` 和既有结果审计器，保留当前 SHA 的原始证据。

## 验证内容

1. 两个应用实例共享 MySQL 与私有对象服务，上传图片在对象层加密，匿名访问被拒绝。
2. 生成真实会话、健康记录、SOS、通知 outbox、事件及访问审计，先完成 PG 投影同步。
3. 在停止应用读写后保存真实 mysqldump、密文对象及独立目录中的测试密钥。
   对迁移历史、健康、工单、事件、审计、通知队列和会话计算有序记录摘要，不输出记录内容。
4. 实际删除测试主库、对象及 PG 投影 schema，清空配置中的密钥；用不同密钥不能解密图片。
5. 恢复 SQL、对象和独立密钥，逐类比较摘要；运行当前迁移验证，迁移历史不得变化。
6. 新应用实例能用恢复的会话读取原图片；由恢复的 MySQL 重新 backfill PG，
   比较 source_id、entity、entity_id、payload、deleted 的逻辑内容。
   不比较本次重新投递的时间和事件版本。重复重建必须幂等。
7. 恢复后重试相同 requestId 的 SOS，不增加工单或通知 outbox 条数。

## 证据边界

这是小规模自动化灾损回归，测试进程内临时文件承载测试密钥；不是生产 KMS、异地保管、
密钥退役或生产备份作业的认证。没有测定生产数据规模下的 RPO/RTO、长时间停机、
外部通知送达或备份期间持续写入的保证。这些项目仍需独立演练与记录。
恢复失败时不输出可能包含敏感值的 SQL dump 或 SQL 错误上下文。
