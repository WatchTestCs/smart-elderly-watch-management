# 本地耐久浸泡：本日全部优化后的稳定性闸门（2026-10-24）

代码 `security-fixes` 分支（含本日全部提交：会话单往返、限流分片、响应包络、
观测中间件、analytics 租约、规则引擎 memo，及依赖升级 go1.26.10/pgx 5.9.2）。

## 执行

`SOAK_MYSQL=1 SOAK_DURATION=10m go test -run TestMySQLTwentyMinuteSoak ./internal/handlers`
（CI 同款 20 分钟档的 10 分钟本地档）。隔离 MySQL 8.4（Docker，loopback）、
4 工作协程、批 200 样本、真实 SaveBatch+commit 与规则评估回路。
原始摘要：[soak-summary.json](soak-summary.json)。

## 结果（PASS）

| 指标 | 值 |
| --- | --- |
| 请求 | 1,173（全部 HTTP 200；0 请求错误、0 5xx、0 规则错误） |
| 持久化 | 234,600 样本 = 每秒 391 样本，无丢失无重复 |
| 延迟 | p50 39.0ms / p95 64.6ms / p99 161.9ms / max 4,638ms（启动预热期） |
| 堆 | 稳定于 2.4–3.9MB，10 分钟无增长趋势（无泄漏迹象） |
| GC | ~94 周期/分钟（默认 GOGC；未开 AUTOTUNE） |
| 连接池 | in_use ≤1、等待 0 次 0 毫秒（20 连接上限远未触及） |

每分钟明细见 JSON `per_minute`；该测试的固有局限（单 httptest 处理器、合成
数据、无外网投递）见 JSON `limitations`。

## 意义与边界

- 在持续写入负载下验证了本日全部热路径改动（含会话/csrf/auth 合并查询、
  限流分片、响应零拷贝包络、观测标签驻留）与租约式 analytics 消费无错误、
  无连接耗尽、无内存增长。
- 10 分钟与 4 并发不能替代 CI 20 分钟档与生产观察；作为合入前的本地闸门。
