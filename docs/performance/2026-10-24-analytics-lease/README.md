# analytics 投影租约式消费（2026-10-24）

基线 `cd22f51`（security-fixes 分支）。解决 docs/performance/2026-10-23-hot-path 点名的架构问题："PG 写入期间持有已选 MySQL outbox 行锁和事务连接"。

## 旧事务形状与问题

`SyncBatch` 在单个 MySQL 事务内 `SELECT ... FOR UPDATE SKIP LOCKED` 锁定 ≤500 行，**保持事务开启**的状态下执行 PG UNNEST 写入，成功后才 DELETE+COMMIT。PG 慢或不可达时：outbox 行锁与 20 条业务连接池中的一条被占用整个 PG 写入时长（跨网 RTT × 批次数）；PG 长时间故障会持续吞噬业务连接。

## 新三段式（迁移 mysql/0012：`analytics_outbox` 增加 `lease_token/lease_until` 可空列，触发器 INSERT 不受影响）

1. **认领（短事务）**：SELECT 过滤"无租约或租约已过期"并 `FOR UPDATE SKIP LOCKED` 锁行 → 同事务写 `lease_token=uuid, lease_until=now()+30s` → **提交即释放全部行锁**。
2. **投递（零 MySQL 持有）**：事务外执行与旧实现逐字相同的合并+UNNEST 幂等 upsert（`last_event_id` 拒旧）。此阶段不持有任何 MySQL 行锁或连接。
3. **确认（短事务）**：`DELETE ... WHERE lease_token=? AND id IN (...)` 只删仍归本租约的行。

故障语义：
- **崩溃**（认领后未确认）：租约 30s 到期，下一轮重新认领重投；PG 端 `last_event_id` 幂等保证不产生重复（与旧实现"PG 先行提交、MySQL 后确认"的崩溃重放论证同源）。
- **PG 写失败**：立即 `releaseLease`（token 清空）归还事件，恢复旧实现的快速重试节奏，不必等 30s。
- **多工作者**：SKIP LOCKED + 租约 token 双保险；过期被他人重新认领的行，旧持有者的 DELETE 因 token 不匹配不会误删（返回 0）。

## 证明与测量（真实 MySQL 8.4 + PostgreSQL 16）

- `TestClaimReleasesRowLocksImmediately`：认领提交后，另一连接对同一行 `FOR UPDATE NOWAIT` **立即成功**——PG 写入尚未开始行锁已释放，这是核心收益的机制性证明（旧实现此时持锁直至 PG 写完）。
- `TestCrashAfterClaimReplaysOnceAndIdempotent`：认领后"死亡"、强制租约到期 → 重投 → outbox 清零、PG 恰一条正确记录。
- `TestAckOnlyDeletesOwnLease`：过期重认领后，旧 token 确认删除 0 行，行由新持有者持有。
- `TestReleaseLeaseOnApplyFailure`：PG 不可达 → SyncBatch 失败且租约立即归还 → 马上可重新认领。
- 既有全部测试不改一行通过（事务性 outbox 语义、目的地中断不确认、恢复重投、墓碑 NULL、积压清空、消费/生产 A/B、往返 A/B）。
- MySQL 锁窗口：旧 = 认领+PG 写+确认全程；新 = 仅认领与确认两个短事务（各毫秒级）。吞吐按批不变（同等往返次数：认领/确认两次 MySQL 往返 + 一次 PG 往返，与旧的 SELECT/DELETE/COMMIT 等量）。

## 边界

- 每批事件在认领与确认间有 ≤30s 的重复投递窗口（崩溃场景）；PG 幂等使其不可见，MySQL 侧事件不丢不重。
- `SyncOne`（测试用单事件路径）保留旧形状未改。
- 未做：认领与投递的流水线并行（下一批认领可与上一批 PG 写重叠）——Run 循环为串行 drain，收益需先测 PG 写入占比，列为后续门槛。
