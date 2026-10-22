# MySQL SQL 往返、分析积压与日缓存写入性能优化（2026-10-22）

**目标：** 在不删触发器、不删历史索引、不改业务事务语义的前提下减少 SQL 往返、后台同步空转和无意义的缓存更新。

## 改动范围

1. **MySQL 参数插值（默认启用）**。业务主库固定使用 `utf8mb4`，启用官方 `go-sql-driver/mysql` 的 `InterpolateParams`，让单次参数化 `Exec/Query` 不再每次执行 `PREPARE + EXECUTE + CLOSE` 的额外通信往返。仍使用参数绑定接口，不允许应用自行字符串拼接。可将环境变量 `MYSQL_INTERPOLATE_PARAMS=0` 回退到原 prepared-statement 路径；只接受空值、0/1 或 true/false（大小写不敏感）。
2. **分析队列饱和排空**。原同步循环最多处理 10×250 个事件后总是睡 2 秒。在连续完成 10 个完整批次时改为短暂 10ms yield 后立即继续；空闲、短批次和故障重试仍休眠 2 秒，保留批次数、事务边界与取消限制。
3. **队列深度状态采样**。`SELECT count(*) FROM analytics_outbox` 从每轮（约每 2 秒）改为每 30 秒至多一次、最多占用 2 秒；统计错误不再阻断 PG 同步。状态字段 `pending` 现在是**最近一次精确采样扣除本进程已消费量的估计**，可能在 30 秒内暂时低估并发写入；不用于判断通知送达或执行关键业务。
4. **日统计缓存写入**。批量健康样本提交时，只对 `old_id` 相同且 `dirty=false` 的缓存项执行 `SET dirty=true`。既有脏记录无需再次写入或保持冗余排他锁；仍覆盖该老人所有日期与来源，保留更正／删除的保守失效策略。

## 测量：MySQL 8.4、Go 1.23.2、临时数据库

以下为每轮 ABBA 顺序的两次样本平均吞吐（不是生产 TPS）；批量写入测试保持原 `ratepre` 触发器，逐条写入每事务 1 行，批量每事务 100 行。两个驱动模式使用相同 SQL 参数、MySQL 会话选项与数据结构，回读校验全部源行和 Outbox 行。

| 操作 | [运行 39755296645](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39755296645) 插值 / 原模式 | [运行 39755682111](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39755682111) 插值 / 原模式 |
|---|---:|---:|
| 单条参数化 MySQL 主键查询 | **1.94×** | **2.00×** |
| 每事务 1 条业务写入 | **1.52×** | **1.52×** |
| 每事务 100 条业务写入 | **1.14×** | **1.26×** |

最早的 [运行 39754525915](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39754525915) 也记录了查询约 1.98×、单条写入约 1.21×，但 100 条写入的插值组出现一次严重离群（约 1,755 行/秒），该轮批量均值反而下降。**报告完整保留这一异常，不能声称所有负载都获得加速。** 实际生产表现仍取决于磁盘、连接并发、CPU、网络距离与业务行宽。

### 队列深度查询成本

[第三轮 A/B 日志](run-39755682111.jsonl) 的单次查询 16 次采样显示：

| Outbox 积压 | 精确 `COUNT(*)` p50 | `EXISTS` p50 |
|---:|---:|---:|
| 0 | 0.159 ms | 0.153 ms |
| 10,000 | 0.670 ms | 0.245 ms |
| 100,000 | 3.162 ms | 0.644 ms |
| 500,000 | 14.167 ms | 0.834 ms |

出于 UI 需要仍保留精确 `COUNT(*)` 采样，不把 `EXISTS` 冒充队列精确总数；仅将默认每 2 秒一次的采样频率降低至每 30 秒一次（最高约减少 93% 的状态计数次数）。

### 饱和同步上限

旧调度固定每次最多 `250 × 10 = 2,500` 个事件、随后无条件延迟 2 秒，因此**持续饱和的理想上限约为 1,250 个事件/秒**，即使单批 SQL 更快也会遭受调度限速。新调度只在连续完成全部 10 个满批次时立即继续，使处理能力由数据库和配置决定。该改动没有宣称完整服务已达 30,000 事件/秒；必须参考 `TestRunnerDrainsSaturatedTransactionalBacklog` 真实双库端到端排空日志、并进行生产拓扑验收。

## 验证与证据

- [初次 SQL 往返原始数据](run-39754525915.jsonl)
- [复测＋队列深度原始数据](run-39755296645.jsonl)
- [再测＋队列深度原始数据](run-39755682111.jsonl)
- GitHub Actions 原始 job 日志与 artifact 仍是第一手可追溯证据。
- 默认 CI 的 Go 1.23.2、Go 1.27.2、race、build、vet、gofmt 和临时 MySQL/PG 集成测试必须成功后才可合入。
- `TestSaveBatchOnlyInvalidatesCleanDailyCacheRows` 验证事务回滚不泄露缓存失效、仅对应老人缓存标记为 dirty。
- `TestRunnerDrainsSaturatedTransactionalBacklog` 从真实 MySQL 触发器生成 6,000 个事件，启动实际同步工作循环，在独立 PG 库检查完整投影和 Outbox 清空。

## 部署、观测、回滚

- 修改不需要新的 MySQL/PG 迁移，不移除任何触发器、索引或 Outbox 持久化约束；原应用启动流程不变。
- 生产启用前在实际网络拓扑压测 `MYSQL_INTERPOLATE_PARAMS=1`（默认）与 `=0` 的写入 p95/p99、连接池等待、DB redo、CPU、PG 积压和通知延迟，监测多实例并发。
- 如发现参数处理与延迟异常，**设置 `MYSQL_INTERPOLATE_PARAMS=0` 并重启服务**恢复原方式；对于队列循环改动需回滚对应代码提交。
- `pending` 是**近似诊断值**。若需要严格对账，使用授权的 MySQL 独立只读查询及一致性流程；不能以该字段作为 SOS 通知、已送达或已有人接手的承诺。
- 本轮不调整 MySQL `innodb_flush_log_at_trx_commit`、`sync_binlog` 或会丢数据的优化开关，也不做主库自动切换。
