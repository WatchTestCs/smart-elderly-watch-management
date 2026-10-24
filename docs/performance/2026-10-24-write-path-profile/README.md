# 健康批量写入分阶段画像（2026-10-24）

执行 p99 文档列出的后续门槛：量化同 owner 饱和场景下单批服务时间的真实分布。
工具：`internal/handlers/health_profile_test.go`（opt-in `HEALTH_PROFILE=1`），
对真实 MySQL 8.4 依次计时 handler 的每个阶段（20 次迭代，200 样本/批，
含既有数据多版本场景）。

## 结果（本机 Docker MySQL，loopback）

| 阶段 | 平均 | 占比 | max |
| --- | ---: | ---: | ---: |
| 设备鉴权 + 行锁 | 0.87ms | 2.5% | 1.24ms |
| SaveBatch 全部语句（owner 锁/查找/预留/双 INSERT/脏标记） | 18.75ms | 54% | 33.2ms |
| **COMMIT** | **15.37ms** | **44%** | 32.2ms |
| 合计 | 35.0ms | | 47.9ms |

## 结论

1. **写入路径已 fsync/DB 主导**：最大单项是事务提交的 InnoDB fsync（44%），
   属持久性语义，代码层不可削减；其余为语义必需语句（owner 时钟锁+预留
   两段是修订号顺序正确性所需，批语句已充分批量化）。
2. 这解释了 2026-10-24-p99-rig 的观察：饱和 p99 由 fsync × 串行化队列
   决定，本日 CPU 优化在该场景被吞没；代码层优化战役已到测量地板。
3. 剩余合法杠杆在部署侧（如 `innodb_flush_log_at_trx_commit` 的持久性
   取舍、磁盘 fsync 性能、组提交并发度）——属运维决策，非代码改动。

## 复现

```bash
HEALTH_PROFILE=1 TEST_MYSQL_DSN=... \
  go test -v -run TestHealthBatchPhaseProfile ./internal/handlers
```
