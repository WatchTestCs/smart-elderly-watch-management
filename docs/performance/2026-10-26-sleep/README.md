# 睡眠统计内存与排序优化 · 2026-10-26

基线为上一轮已提交的 `preview@d2c70eb33b7fce18ed330358a5eac279d3f7d964`。[上一轮报告](../2026-10-26-algorithms/README.md)保留其全部实验；该基线的 [Preview CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/40283334762)、[Security](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/40283334750) 和 [UI Quality](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/40283334725) 已通过。这些检查对应基线，不自动成为本次新代码的验收。

## 问题与实现

上一轮消除了最终区间合并副本，但睡眠统计仍把相同记录分别复制到各阶段、睡眠和清醒的区间数组，并分别排序。10 万条样本的聚合调用仍分配约 55.9 MB；HTTP 统计与后台日统计均调用该算法。

本次复用 Aggregate 已持有的私有样本指针数组，先原地压紧有交集的有效区间，再使用 Go 标准库 `slices.SortFunc` 按起点统一排序。一次遍历同时更新各阶段、睡眠、清醒以及两者并集的累计状态，不再建立这些区间数组。最终仍用“睡眠分钟 + 清醒分钟 − 两者并集”计算冲突。

排序只移动临时指针，调用者的 Sample 数组与字段保持原样。将起点裁到报告下界不会破坏已排序次序；裁剪产生的同起点区间相互重叠，其先后不改变并集端点。空窗口、逆序窗口、未知阶段和全部在窗口外的记录保留旧函数的结果与 Count 语义。窗口外记录在排序前筛除，避免对本无需排序的数据引入 O(n log n) 开销。

整体时间仍为 O(n log n)，临时存储仍为 O(n+k)，其中 n 为有效样本数、k 为阶段种类；本轮减少的是重复排序和区间副本。不是零分配，也没有降低时间精度、删减冲突或改成近似算法。旧区间类型与并集合并代码从生产路径删除，完整固定基线 oracle 留在测试文件中。

## 交替基准

AMD EPYC 9V74 80-Core Processor，Linux amd64、Go 1.23.2，cgroup CPU 配额 `800000 100000`。共享运行环境，未绑核。相同夹具依次 A1 → B1 → B2 → A2，各项 200ms，GOMAXPROCS=1/4；共 54 个场景与 P 的组合。下表是单 P 两轮算术平均，耗时单位 ms/op。

| 场景 | 基线 → 最终 ms/op | 耗时变化 | 分配 B/op：基线 → 最终 |
| --- | ---: | ---: | ---: |
| 10 万条重叠睡眠 | 70.140 → 30.701 | -56.2% | 55,874,144 → 4,496,952 |
| 10 万条互不重叠睡眠 | 69.653 → 35.484 | -49.1% | 55,874,144 → 4,496,952 |
| 10 万条全部在窗口外 | 5.070 → 4.274 | -15.7% | 4,496,552 → 4,496,552 |
| 10 万条仅 1 条有效 | 0.170 → 0.151 | -11.4% | 408 → 328 |
| 1,000 条睡眠 | 0.130 → 0.071 | -45.3% | 262,408 → 17,848 |
| 10 条睡眠 | 0.002 → 0.001 | -49.9% | 3,976 → 568 |

10 万条重叠/互不重叠场景的分配均下降约 **92%**（55,874,144 → 4,496,952 B/op）。这些数字只计聚合调用的分配，不包括预先建立的 Sample 数组、数据库返回结果、JSON 解码或进程 RSS。没有直接测量 GC 暂停、生产峰值内存、数据库锁等待或 HTTP p99，不能据此宣称这些指标改善了同样比例。

保留所有控制场景。下列非睡眠场景出现超过 5% 的耗时回退；对应算法未改写，但不能仅凭两轮数据把变化归因于环境：

| 控制场景 | 耗时变化 |
| --- | ---: |
| `BenchmarkAggregateLarge/daily_total/10000-4` | +9.2% |
| `BenchmarkAggregateLarge/instant/100000-4` | +6.5% |
| `BenchmarkAggregateLarge/deleted/100000-4` | +6.0% |
| `BenchmarkAggregate/instant/1000` | +5.5% |
| `BenchmarkAggregate/sparse/1000-4` | +5.6% |

没有修改、筛掉这些结果，也没有声称所有路径都更快。完整均值与分配见 [summary.json](evidence/summary.json)，四轮完整输出及源码/夹具散列见 [environment.json](evidence/environment.json) 和同目录日志。每份基准输出均检查 54 行结果及 PASS 后保存。本轮不与上一轮不同时间测得的绝对延迟直接拼接成生产提升倍数。

## 验证

- 新增 1,200 组睡眠差分测试，与固定 `8c22299` 原函数和原区间并集实现比较结果、错误及浮点位。覆盖同起点、相接、嵌套、跨窗口、空/逆序窗口、无效原始区间、未知阶段、日累计选择、删除与缺值，并检查调用者数据不变。已有 3,000 组混合指标差分继续执行。
- 最终 `go test -race -short -json ./... -count=1 -timeout 180s`：**891 项测试/子测试通过，125 项跳过，0 失败**。`go build ./...`、`go vet ./...` 通过。输出与逐项跳过原因见 [validation.txt](evidence/validation.txt)。本机未配置 MySQL/PG 测试连接；数据库集成及 opt-in 用例不算通过。
- 本轮只修改健康统计实现、测试和文档，没有 SQL、迁移、鉴权、队列认领或前端行为变更；没有重复运行不相关的前端全套检查。新提交的真实数据库 CI 与设备/原生壳验收应按对应提交核对。

## 复现与剩余瓶颈

两个独立工作副本分别检出上述基线与候选；将候选的 `internal/health/statistics_sleep_test.go` 和更新后的 `statistics_stream_test.go` 原样复制到基线同路径。从两个 backend 目录交替执行：

```sh
go test ./internal/health -run '^$' -bench '^Benchmark(Aggregate|AggregateLarge|SleepWindowLayout)$' -benchmem -benchtime=200ms -count=1 -cpu=1,4
```

Sample 数组与 JSON 解码仍可能占据较多内存，后台日统计仍按原 owner 锁串行维护，数据库读取及锁持有时间尚未重测。本轮不通过扩大并发或放宽事务确认去掩盖这些成本；后续若调整流式解码/后台调度，需单独验证错误优先级、100,000 条上限、并发失效与落库一致性。
