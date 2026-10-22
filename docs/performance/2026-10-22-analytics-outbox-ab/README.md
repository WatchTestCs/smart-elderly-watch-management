# MySQL 触发器与分析投影 A/B 验证（2026-10-22）

## 结论与合并决策

**保留现有 MySQL 分析 Outbox 触发器，不合并 `0010_mysql_schema_and_index_optimization.sql` 中的删触发器命令。** 并非假定触发器零成本，而是在相同“主库业务数据 + 可恢复 Outbox 事件”输出下进行了两次隔离测量：

- **逐条业务写入**：触发器方案平均吞吐在两轮运行中分别是显式事务 Outbox 方案的 **1.38 倍**和 **1.49 倍**；移除触发器反而较慢。
- **每事务 100 条写入**：事务内批量生成 Outbox 的替代方案在两轮中分别比触发器高 **13.8%**和 **8.2%** 的业务写入吞吐；该幅度尚不足以独立证明全链路迁移收益，更不能替代安全验收。
- **MySQL → PostgreSQL 投影消费**：已在 [PR #22](https://github.com/WatchTestCs/smart-elderly-watch-management/pull/22) 合并的批量消费相对旧单条消费，在两个测试运行中分别达到 **59.9 倍**和 **57.8 倍**的测试吞吐。其加速点在同步消费者，不要求拆除触发器。

这意味着当前优先优化方向应为 **保留事务内 CDC，批量消费 Outbox**。不把样例微基准吞吐推导成生产的总体 TPS 或 SLA。

## 两次实测（每项两轮 ABBA、相同输出）

以下为每次 GitHub Actions 内同一模式两个 trial 的平均值，单位条/秒；两次运行在不同 GitHub 托管环境中执行，不能合并为同一台固定规格服务器上的统计置信区间。

| 测试维度 | [运行 A](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39750285619) | [运行 B（全量 JSON 校验）](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39750729750) |
|---|---:|---:|
| 消费：单事件 `SyncOne` | 505.98 | 470.62 |
| 消费：批量 `SyncBatch` | 30,299.26 | 27,185.19 |
| 写入：每事务 1 条，触发器 | 796.66 | 754.06 |
| 写入：每事务 1 条，应用事务内 Outbox | 577.43 | 504.98 |
| 写入：每事务 100 条，触发器 | 18,604.65 | 18,275.17 |
| 写入：每事务 100 条，应用事务内 Outbox | 21,166.41 | 19,766.76 |

完整单次数据见 [首轮原始 JSONL](results.jsonl)、[全量摘要校验复测 JSONL](results-verified.jsonl)。保留实际 `elapsed_ms`、`rows_per_second`、`transaction_p50_ms`、`transaction_p95_ms`、`transaction_p99_ms` 和已投影/待处理数，不仅存档均值。原始日志在对应 Actions 的 `analytics-outbox-ab` Artifact 中。

## 实验方法

- 环境：GitHub Actions Ubuntu runner，**MySQL 8.4 / PostgreSQL 16 / Go 1.23.2**，随机临时业务库与独立 PG 库；从未连接生产 DSN。
- 生产者：对每次试验建与 `ratepre` 相同索引的 `ratepre_direct` 表（`CREATE TABLE … LIKE`，**不继承触发器**）；对照组使用原 `ratepre` 及原 AFTER INSERT 触发器。替代组在**同一 MySQL 事务**中先批量写 `ratepre_direct`，再用一条 `INSERT INTO analytics_outbox … SELECT … JSON_OBJECT` 批量写同等事件；计时涵盖事务创建、SQL 和提交，不包含数据清理。每事务 1 条模式测试 1,200 条；每事务 100 条模式测试 6,000 条。采用 ABBA 次序降低初末漂移的影响。
- 数据一致性：业务行数、Outbox 行数均核对；复测逐条按 `entity_id` 排序、规范化全部 JSON 并计算 SHA-256。两种生产方式对应批量下摘要一致；两个样例摘要记录于 `results-verified.jsonl`。
- 消费者：先由实际触发器产生 1,200 条事件，在同一组 MySQL/PG 中分别用原 `SyncOne` 与现 `SyncBatch` 完整消费；测量不包括数据生成、投影表清理和连接初始化。均核对 ACK 数、PostgreSQL 实体数、待处理事件数及末项数值。两个模式各运行两轮（ABBA）。
- 完整 CI 测量由 [专用工作流](../../../.github/workflows/analytics-outbox-ab.yml) 执行；普通服务 CI 不自动运行高负载 A/B。

复现（**只允许指向可创建/清理随机数据库的测试实例**）：

```sh
cd 智慧养老手表管理系统/backend
export RUN_ANALYTICS_DB_AB=1
export TEST_MYSQL_DSN='测试MySQL原生DSN'
export TEST_DATABASE_URL='测试PostgreSQL连接URL'
go test -run '^TestAnalytics(ProducerAB|ConsumerAB)$' -count=1 -v -timeout 18m ./internal/analytics
```

## 限制与高风险行为

1. **只模拟 `ratepre` INSERT**。尚未比较三张遥测表的真实混合 `INSERT/UPDATE/DELETE`、通知/规则并发、断网重试、多实例、真实设备及历史回填。
2. 替代生产者使用 `INSERT … SELECT` 在数据库事务内重新读取业务行；这不是已部署的 Go 应用层实现，也不是已完成的业务写入替换。不同 SQL、批量大小、行宽及部署拓扑会显著影响结果。
3. 事件 JSON 内容与条数相等，只证明这些样本的内容匹配；真正替换前仍需验证提交/回滚原子性、删除墓碑、源端直接 SQL 写入和投影恢复，不得依据纯吞吐先删除触发器。
4. 复测并未进行长时间负载、CPU/RSS、redo 字节、主从复制、p99 整条 SOS 链路及告警送达测量；两个环境每个场景只各做两轮，不足以形成统计显著性结论。
5. `analytics_outbox` 的主键已支持 `ORDER BY id LIMIT … FOR UPDATE OF o SKIP LOCKED`；原补丁新增的 `(id,entity,entity_id,deleted)` 索引并未证明能改善扫描，且会增加写维护。原补丁声称将生成列 STORED 改为 VIRTUAL，但现有 `0009_rule_baseline_cover.sql` 已是 VIRTUAL；此外原补丁错误改用了 `$.val` / `$.legacy_unzoned`，直接执行会改变结果含义。
6. 原补丁的 `idx_health_rule_window` 仍是现有缺失指标规则查询的计划预期，不能凭“索引多”就删除。

## 退役触发器的必要前提

只有在能锁定所有写入来源、每个新增/修改/删除都能在同一 MySQL 事务中产生等价 Outbox 事件，并通过崩溃、回滚、跨进程、同步失败及回填对账验证后，才可考虑**受控切换**。禁止“先删后补”，也不允许触发器和应用端同时无去重地写入导致事件翻倍。切换后仍需在相同硬件和业务混合负载下记录 p50/p95/p99、数据库 CPU/redo、PG 积压和投影一致率，再作是否删除的判断。

因此，本轮结论是**对既有消费优化补充实测证据，保留触发器，拒绝原补丁危险 DDL**。
