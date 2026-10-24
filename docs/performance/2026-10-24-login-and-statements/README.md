# 登录路径画像与批语句逐条计时（2026-10-24）

补齐剩余未探测的延迟面。两个 opt-in 工具入库：
`internal/handlers/login_profile_test.go`（`LOGIN_PROFILE=1`）、
`internal/health/statement_profile_test.go`（`HEALTH_STMT_PROFILE=1`）。

## 登录（真实 MySQL，24 次 HTTP）

| 指标 | 值 |
| --- | --- |
| 登录全程 p50 / p95 / max | 247.9 / 280.9 / 283.0 ms |
| BCrypt cost-12 单次 | 203.0 ms（**占 82%**） |
| 裸往返（SELECT 1） | 0.04 ms |

**结论**：登录延迟由设计主导——BCrypt cost 12 是抗爆破约束，DB 部分
仅数 ms（session INSERT + audit INSERT + 两次 SELECT，裸往返 0.04ms）。
降它只能降安全强度，拒绝。首屏体验的合法杠杆是前端（登录页已懒加载）。

## 批语句逐条（1000 行既有样本，15 轮回滚式）

| 语句 | 平均 | max |
| --- | ---: | ---: |
| 1 owner 锁 | 0.74ms | 1.17ms |
| 2 lookup（200 id IN 主键） | 1.53ms | 2.29ms |
| 3 revision 预留 | 0.97ms | 1.51ms |
| **4 upsert（200×14 参数）** | **5.20ms** | 7.43ms |
| 5 events（200×4） | 2.48ms | 4.87ms |
| 6 daily 脏标记 | 0.55ms | 1.51ms |
| 合计 | 11.47ms | |

与 SaveBatch 总量 18.75ms 对账：差额 ~4ms 为样本准备 CPU（digest+marshal，
本日已优化）与行传输/Scan。**最大单条是 200×14 参数的 upsert（5.2ms）**——
已是最小语句形态（单语句多行），InnoDB 写放大是物理成本。写入路径
"代码层地板"结论获得逐条证据背书（fsync 44% + 语句 54% 的分布不可再压缩）。

## 边界

- 本机 Docker loopback；生产磁盘的 fsync 占比可能更高，语句占比更低。
- 两个画像工具均为 opt-in 门控，不进常规测试路径。
