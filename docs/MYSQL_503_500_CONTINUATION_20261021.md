# Go 后端 500/503 接续排查记录（2026-10-21）

## 已核实的工程事实

- 用户上传的 `upgrade.zip` 包含完整 Go 后端、MySQL 迁移与历史规则测试；该压缩包中的 `outputs/` 只有规则测试与内存 CPU 基准。截图中后来运行的 20 分钟 HTTP/MySQL 压测日志不在此包内，**不能据此宣称已经复现或排除所有 500/503**。
- `cmd/server/database.go` 原先每 3 秒调用 `Database.Ping`，其中 MySQL `Ping` 使用了 20 连接的业务池；连续健康检查超时会直接将整个 API 下架为 503。业务池被占满时，即使数据库仍然可用，也有可能发生这种误判。
- 规则引擎 `Device` 在 `SELECT ... FOR UPDATE` 取得设备行锁之后查询最近值、7 天个人基线、活动窗口，并持有锁到状态写入及通知任务入库。其整体耗时随数据规模、连接等待和 SQL 执行计划变化。
- 已有 `migrations/mysql/0006_health_performance.sql` 创建索引 `idx_health_rule_window(old_id,device_id,metric,deleted,sample_start,revision)`。本轮**未盲目增建索引**，也没有调整规则阈值、7 天基线、事务锁或通知语义。

## 本轮代码更改

1. `internal/db/database.go`：开启 MySQL 时另外创建**一个**独立的最大连接数为 1 的健康检查池；`Database.Ping` 使用该预留池，不再排队等业务池的 20 个连接。独立连接不会参与任何健康数据读写或规则事务。若任一池初始化失败，两个池按需关闭并保持拒绝启动；正常关闭同时释放两个池。每个应用副本最多增加一个 MySQL 连接，务必核算数据库 `max_connections`。
2. `cmd/server/database.go`：授权 `/metrics` 增加业务池 `OpenConnections/InUse/Idle/MaxOpenConnections/WaitCount/WaitDuration`、健康探测成败指标。没有记录患者、设备、密码或完整 SQL。
3. `internal/rules/engine.go`：支持环境变量 `RULE_DEVICE_WORKERS=1..8` 调整每副本最多同时持有的设备事务数；默认保持原 8，不改变生产部署原有行为，越界值回退为 8。应在同样负载下对照 1/2/4/8，不能单凭并发上限推断吞吐提升。
4. 新增基于真实 MySQL 的回归测试：业务池仅允许 1 连接且被占满时，健康探测仍应成功。未连接到测试 MySQL 之前，该测试**不能视为已通过**。

## 验证结果与限制

- 结构验证：142 个 `.go` 文件经 Go 1.23.2 标准库 `go/parser` 全部解析，语法错误 0。
- 隔离验证：规则引擎并发环境变量解析单测在 Go 1.23.2 下 `-race`、`go vet` 通过。
- 完整后端：**未执行成功**，上传源码 `go.mod` 要求 Go 1.24.0 或更新；当前执行环境 Go 1.23.2，外部依赖未缓存、没有 MySQL 实例；完整 `go test` 明确报告版本不兼容。不要将结构验证当成完整编译或数据库验收。
- 当前没有真实 20 分钟 HTTP 压测日志，**无新的 HTTP 503、500 次数、p95/p99、写入完整性或性能增益结论**。

## 恢复环境后的顺序

1. 在 Go 1.24+、MySQL 8.4 一次性测试实例上执行：
   ```bash
   cd 智慧养老手表管理系统/backend
   go test ./... -count=1
   TEST_MYSQL_DSN='测试专用MySQL DSN' go test -race ./internal/db ./internal/rules -count=1
   ```
   `TEST_MYSQL_DSN` 必须有 CREATE/DROP DATABASE 权限，仅用于隔离实例，绝不能指向生产实例。
2. 读取 `/metrics`，观察 `suining_mysql_health_probe_failures_total` 和 `suining_mysql_pool_wait_total` 的增量；同时观察 `in_use` 是否逼近 `max_open_connections`。比较新旧版本在**业务池被占满但 MySQL 仍活着**时 `/restful/service/status` 和实际 HTTP 状态，确认不因探测排队产生误报 503。
3. 在测试环境分别以 `RULE_DEVICE_WORKERS=1,2,4,8` 运行同一套健康数据写入、规则扫描、通知任务负载。记录请求总数、2xx/4xx/500/503、落库行数、重试/重复、MySQL 事务等待、死锁、p50/p95/p99、RSS、GC，并比对事务及告警结果一致性。
4. 使用同样数据规模的真实负载查询执行计划：
   ```sql
   SHOW INDEX FROM health_sample;
   -- 从你现有带参数日志/测试驱动中对基线聚合 SELECT 执行 EXPLAIN ANALYZE。
   -- 确认 idx_health_rule_window 的访问行数和实际执行时间，
   -- 并分离 JSON_EXTRACT、锁等待、连接等待与网络层超时。
   SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT/1000000000000 AS total_seconds
   FROM performance_schema.events_waits_summary_global_by_event_name
   WHERE EVENT_NAME LIKE 'wait/lock/%'
   ORDER BY SUM_TIMER_WAIT DESC LIMIT 20;
   ```
   上述 Performance Schema 查询需要对应权限，不同 MySQL 配置下数据可缺失。不要在生产高峰执行未限额的 `EXPLAIN ANALYZE`。
5. 补齐截图中的原始 20 分钟压测日志，尤其是最后发生 500 的应用异常日志、MySQL 慢查询/执行计划、500/503 的时间戳，以及每批提交与持久化记录数量。没有这些证据就不能把 500 归因为单一 SQL 或宣称修复。

## 不变的安全约束

- 不自动重试非幂等写入，不因瞬时网络失败重复发出 SOS 或重入通知；必须通过持久化唯一键/出站队列保证幂等。
- 不关闭或削弱超时与错误校验；数据库短时真实不可达时仍按服务设计返回错误，不伪造“已通知/已入库”。
- 保持鉴权和隐私日志边界；真实数据库数据、DSN 密码、密钥不能随交付打包。
