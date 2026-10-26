# 本机性能与内存证据

这里全部是本机 microbenchmark 或 loopback HTTP benchmark，不是 staging/production。
没有改事务、审计、响应前持久化、数据库 durability 或照护状态机来改善数字。

## 统计与锁

50b9cc0 与 110b5a4 的 `backend/internal/health` tree ID 均为
`a5d47ffd4610e483ef07b0171d84f5c033a544a5`，统计实现未改。
两历史基线与候选在真实 MySQL/PG/race 下通过；覆盖日统计聚合、失效、owner clock 并发独立提交、
rollback 与历史版本、睡眠差分等。CI 原始 JSONL 可按 TestAggregateDailyCalendarEquivalent、
TestSaveBatchOnlyInvalidatesCleanDailyCacheRows、TestOwnerClockAllowsIndependentCommits 等查找。
这证明功能/并发回归，不代表生产后台任务下的 tail latency 已测完。

## HTTP p95/p99

固定合成 10000 健康样本，真实 MySQL 8.0.43/NVMe，Go 1.26.10/Windows；走认证、数据库
session、CSRF、限流、日志/metrics 和真实 loopback HTTP。缓存关闭指应用统计缓存，DB buffer
pool 仍热。基准不含外层 TLS/proxy/后台 worker。全部响应内容和跨实例 session 撤销校验通过。

先跑原有 fixture 两次（反向顺序重跑），原始每次 latency 保留于 http-audit-*。冷路径每轮只有
40 请求，p99 易受调度噪声影响；一轮候选偏慢，另一轮两者差距变化，因此追加等量采样。
追加只通过 Go test overlay 把冷路径 40 改为每轮 200，3 轮，GOMAXPROCS=4；业务源码不变，
补丁和 harness hash 保存在 http-tail-*/，先候选后基线。下表为三轮分位数的中位数，单位 ms：

| 路径 | 50b9cc0 p95 | 110b5a4 p95 | 50b9cc0 p99 | 110b5a4 p99 |
|---|---:|---:|---:|---:|
| statistics_hot | 0.611 | 0.624 | 1.028 | 1.038 |
| statistics_uncached | 33.306 | 31.916 | 35.719 | 34.317 |

冷统计各轮 p99：基线 35.719/39.695/33.333，候选 32.863/34.317/34.380。
没有观察到稳定的统计算法退化，也不据此宣称加速。原小样本登录尾延迟存在变化，bcrypt 与
新增登录审计查询的成本保留；不删除跨实例安全检查换 p99。正式容量门槛仍需同机空闲、重复
随机顺序、真实并发/后台日统计和长期负载测试，不能拿这些数字保证生产 SLA。

旧 fixture 的 `baseline_commit=daac...` 是原脚本硬编码的历史标签，`build_settings` 可能为空。
**本次实际源码以相邻 manifest.json 的 source_sha、干净 worktree 与 harness 补丁为准**，
不得把那个遗留字段误读为本次测试提交。原始文件未为美化结果而改写。

## limiter 微压测

limiter-memory-bf33177 使用真实 Limiter、三波各 100000 模拟来源、IPv4/IPv6 混合、256
goroutine。Windows GetProcessMemoryInfo 每 10ms 采样，保存 RSS 与 OS 峰值，区别于 Go HeapSys。
最终 key=8192、槽=8192、容量拒绝=291808；活跃 key 未随 300000 来源增长。
GC 后 live heap 三波约 2.42/2.50/2.60 MiB，进程 RSS 峰值 23.09 MiB。
完整保留内存边界与多 ingress 限制见 abuse-and-monitoring.md。单次 RSS 不是整个业务进程上限。
