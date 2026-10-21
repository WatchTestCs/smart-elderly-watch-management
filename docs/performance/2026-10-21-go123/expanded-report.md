# 岁宁 Go 1.23.2 扩展验证与离线持续负载（2026-10-21）

## 代码来源与验证边界

- 源码基线：用户提供 `upgrade.zip`，此前交付 `suining-go-continuation-source-20261021.zip` 的业务逻辑。此次**不对用户交付的生产源码作修改**。
- 环境：`go1.23.2 linux/amd64`，AMD EPYC 9V74；cgroup v2 CPU 配额 `400000 100000`（4 CPU），内存配额 4 GiB。`GOPROXY=off`，第三方依赖未缓存，不能联网下载 Go modules；没有 MySQL 服务。
- 将只依赖标准库的**真实生产函数**复制进独立模块 `careold-go`，编译并运行原有逻辑测试。健康归一化、统计函数、规则评估、活动窗口、SQL 参数转换与规则并发配置均使用生产函数；隔离 SQL 转换时剥离了依赖 MySQL/pgx 的数据库入口。
- 原项目 `go.mod` 仍要求 `go 1.24.0`，本包不是完整的 Go 1.23 生产兼容构建。`testing.B.Loop()` 属于 Go 1.24 接口，因此只在此包 `*_test.go` 中改写为 Go 1.23 的 `for i := 0; i < b.N; i++`，未改生产逻辑。
- 不包含真实 HTTP、数据库 I/O、通知投递、SOS 或设备模拟入库；无任何实际写入验收。

## 测试运行结果

| 检查 | 实际执行 | 结论 |
|---|---|---|
| 生产健康和规则逻辑 | `go test -race -count=5 ./...`（包括 SQL 转换） | 通过；未报告数据竞争 |
| 健康样本归一化、睡眠/步数统计、单位/边界检查 | 真实生产函数与原项目测试 | 通过 |
| 夜间时区/DST、基线阈值和活动流式聚合 | 原项目测试（活动 34 子场景） | 通过 |
| SQL 缓存与参数绑定 | `go test -race -count=50 -v ./internal/db` | 50 轮均通过（每轮 3 个具名用例） |
| 静态检查 | `go vet ./...` | 通过 |
| 2 秒模拟持续负载竞态检测 | `go run -race ./cmd/coreload -duration 2s -workers 4` | 无 race、内部错误 0 |
| 完整 Go 项目编译 | 需要 Go 1.24 及第三方模块 | **未能完成，不得宣称通过** |
| MySQL 500/503 与数据持久化闭环 | 没有真实数据库与截图对应的 20 分钟原始日志 | **未验证** |

## 生产活动窗口聚合独立对照（3 轮，中位数）

| 样本/语义 | 旧版引用算法 ns/op | 生产流式算法 ns/op | 相对倍数 | 流式分配 |
|---|---:|---:|---:|---:|
| 100 daily_total | 21,608 | 4,939 | 4.4× | 0 B/op, 0 allocs/op |
| 100 interval | 39,375 | 5,404 | 7.3× | 0 B/op, 0 allocs/op |
| 1000 daily_total | 258,189 | 44,673 | 5.8× | 0 B/op, 0 allocs/op |
| 1000 interval | 677,827 | 52,635 | 12.9× | 0 B/op, 0 allocs/op |

旧版对应 1000 样本约 802,324 B / 1,012 allocs（daily_total）或 1,596,985 B / 1,030 allocs（interval）。这里的差异只涉及已解码样本的 CPU 计算，不包含 SQL 取数、JSON 反序列化或端到端延迟。

## 10 秒 CPU-only 负载、GC 对照（4 workers，批长 200）

每次操作对一个有效样本执行 `Normalize`，对同一静态输入执行 `Evaluate`，不提交数据库或出站通知。p99 是**200 次操作组成的批次在用户态的耗时**，不是 HTTP 请求 p99。运行顺序 A/B/B/A 用于检查简单顺序效应。

| 原始文件 | GOGC | 处理次数/s | 每批 p99 | MaxRSS | GC 次数 | 内部错误 |
|---|---|---:|---:|---:|---:|---:|
| `coreload-gogc100.json` | 100 | 426,960 | 5827 μs | 27.9 MiB | 8,828 | 0 |
| `coreload-gogc200.json` | 200 | 783,600 | 2549 μs | 33.3 MiB | 5,386 | 0 |
| `coreload-gogc200-repeat.json` | 200 | 786,620 | 2525 μs | 50.5 MiB | 5,396 | 0 |
| `coreload-gogc100-repeat.json` | 100 | 476,100 | 4891 μs | 37.0 MiB | 9,765 | 0 |

- 本环境中两次 GOGC=200 的吞吐较高且批次 p99 较小，但驻留内存和采样堆峰值不稳定，不能据此配置生产默认参数。
- `max_rss_kib` 来自 Linux `getrusage(RUSAGE_SELF)` 的峰值 RSS；`peak_heap_alloc_sample_bytes` 每约 100 ms 抽样，可能漏掉瞬间峰值；累计 GC 暂停不等于 GC 全部 CPU 开销。
- 10 秒、同宿主顺序运行会受到共享资源抢占、测试数据规模和背景工作影响；这不构成正式 SLA 或任何绝对吞吐保证。

## 500/503 仍待完成的关键验收

1. 在 Go 1.24+ 和 MySQL 8.4 的**隔离测试数据库**重跑完整 `go test ./... -count=1` 及规则/健康/数据库集成测试，勿连生产数据库。
2. 先确认最近一次 SQL 迁移、`idx_health_rule_window` 索引、基线聚合查询 `EXPLAIN ANALYZE` 的真实扫描行数与实际执行耗时。
3. 对 503 使用同负载对照：业务连接池耗尽但数据库仍在线时，监控保留连接探测是否仍成功；数据库真正离线时必须检测失败。避免把健康探测独立当成修复全部 503。
4. 检查规则引擎持有 `device FOR UPDATE` 锁期间的两次基线聚合查询，记录持续 20 分钟压力下的锁等待、SQL 超时和 p95/p99。
5. 比较成功请求数、唯一健康样本、`health_change` 变更记录以及 Outbox ACK，明确故障/重试下的可证明一致性。

## 复现命令

```bash
# 在本独立复现包根目录，Go 1.23.2，无需任何外部模块
GOTOOLCHAIN=local GOPROXY=off go test -race -count=5 ./...
GOTOOLCHAIN=local GOPROXY=off go vet ./...
GOTOOLCHAIN=local GOPROXY=off go test -race -count=50 ./internal/db
GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=4 go test -run '^$' -bench 'BenchmarkActivityWindow$|BenchmarkEvaluate$|BenchmarkNight$' -benchmem -benchtime=300ms -count=3 ./internal/rules
GOTOOLCHAIN=local GOPROXY=off go build -o coreload ./cmd/coreload
GOGC=100 ./coreload -duration 10s -workers 4
GOGC=200 ./coreload -duration 10s -workers 4
```

所有实测轮次原始输出都位于 `logs/`，实际被测试的源码快照位于 `internal/`，SQL 翻译器提取后代码属于测试副本，不是数据库驱动替身。用例覆盖≠全量生产代码覆盖。
