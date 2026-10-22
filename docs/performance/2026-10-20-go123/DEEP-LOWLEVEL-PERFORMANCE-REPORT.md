# Go 1.23 深层性能、GC、内存峰值与尾延迟实验报告

## 范围、环境与结论

- 代码来源：上一轮 `suining-go123-memory-scheduling-20261020.zip`，仅继续修改其中 Go 后端的健康样本 JSON 预分配量，添加独立负载/监控脚本和文档。**没有修改 MySQL、业务状态机、SOS 通知、加密、权限、数据一致性语义。**
- Go `go1.23.2 linux/amd64`，Intel Xeon Platinum 8573C；当前 Linux cgroup v2 `cpu.max=400000 100000`（4 核配额）、`memory.max=4294967296`（4 GiB）。`cpuset.cpus.effective=0-4` 并不等于拥有 5 核 CPU 时间配额。
- 独立生成标准库模块复制**生产 `sample.go`、`sample_fast.go`、`identity.go`**，对每 200 条混合健康样本执行 Normalize、JSON 编码、SHA-256 身份摘要，将结果保留至本批处理结束。
- 混合数据含普通 ASCII、长设备标识、中文标识和有 metadata 的样本。没有 HTTP/MySQL、磁盘刷写、消息发送或真实设备。**所有吞吐都是 CPU-only，不是生产业务 SLA。**
- **经实测，减少对象数本身并不保证更低的尾延迟**。因此取消了试验中导致持续性能回退的样本 JSON 大缓冲区与连续数值槽方案。现在真正进入生产源码的内存变更是将普通 JSON 预分配容量减少 32 字节/样本；复杂数据仍安全地由 append 扩容或退回 encoding/json。

## 生产代码微基准：精细控制分配

同一个典型样本执行 `Normalize + marshalSampleFast + identityDigest`，Go 1.23：

| 版本 | B/op | allocs/op | 结论 |
|---|---:|---:|---|
| 原源码 | 456 | 4 | 基线，3 轮约 1169–1466 ns/op |
| 新源码 | 424 | 4 | -32 B（约 -7%），3 轮约 974–1509 ns/op；单次耗时波动较大，不宣称确定的加速 |

已运行原有 10,000 组 `encoding/json` 逐字节等价对照；特别路径可自动扩容。没有引入 `unsafe`、汇编、JNI、Cgo 或跨请求复用敏感健康数据的池。

**被否决的试验（不进入生产）：**

- 单批次 arena：200 条普通样本的 JSON 分配可从 201 次降至 2 次，但混合数据与持续 GC 场景吞吐/p99 可退化，且过大的 slab 会扩大 RSS。
- 批量连续数值槽：孤立基准分配次数下降，但微基准耗时及若干持续负载的 p99 退化。因此撤回，不牺牲服务质量换单一指标。

## 30 秒闭环饱和实验：GOGC 取舍

每个进程 4 个 Go 工作线程，两个进程依次运行，真实样本预处理代码，批长 200；独立的闭环执行，**没有生产入口排队**：

| 指标 | GOGC=100 | GOGC=200 |
|---|---:|---:|
| 完成批次数 | 181,678 | 258,870 |
| 样本吞吐 | 1,210,509/s | 1,725,775/s |
| p99 服务耗时 | 3.92 ms | 2.49 ms |
| GC 次数 | 8,820 | 4,649 |
| 累计 GC 暂停 | 4,805 ms | 2,524 ms |
| 采样 HeapAlloc 峰值 | 9.3 MiB | 22.7 MiB |
| Linux VmHWM (RSS 高水位) | 15.4 MiB | 48.4 MiB |
| 内部样本处理错误 | 0 | 0 |

**解释：** 此机器上 GOGC=200 显著减少 GC 频率，吞吐和尾延迟信号更好，代价是更高内存。不同进程、不同时间先后运行并非严格对照，操作系统、共享主机干扰仍可能影响结果。30 秒只能称为短期持续测试，不能证明 24 小时稳定性，更不是最坏情况延迟保证。

## 9 秒受控到达速率：发包真实性检查

发包计划：每秒 3,000 批，每批 200 条，4 工作线程，入队缓冲 128。使用 1ms 调度，**把发包器被 CPU 抢占时漏掉的节拍单独记录**；只有 `pacing_loss_pct ≤ 2%`、`dropped=0` 的试验可以用于并排讨论 p99。p99 包含实际排队，不是单纯服务时间。

| GOGC | 完成样本/s | p99 | GC 次数 | 漏节拍 | 有效性 |
|---|---:|---:|---:|---:|---|
| 100（第 1 轮） | 592,477 | 3.24 ms | 1,349 | 329 批 | 有效 |
| 100（第 2 轮） | 592,769 | 3.86 ms | 1,342 | 324 批 | 有效 |
| 200（第 1 轮） | 593,280 | 2.17 ms | 467 | 301 批 | 有效 |
| 200（第 2 轮） | 597,329 | 2.49 ms | 472 | 118 批 | 有效 |
| 200（异常轮次） | 438,094 | 50.48 ms | 353 | **7,277 批** | **无效，保留数据，不纳入直接比较** |

**不能仅凭 p99 好的一轮称稳定达标**。异常轮次说明测试基础设施本身也需要压力余量。大量漏节拍或入队拒绝必须单独展示，绝不能静默当成服务成功。

## 15 秒 GOMEMLIMIT 控制实验（GOGC=200）

| 配置 | 吞吐 | p99 | 采样堆峰值 | VmHWM |
|---|---:|---:|---:|---:|
| `GOMEMLIMIT=16MiB`，4 worker | 1,694,118/s | 2.55 ms | 9.0 MiB | 14.0 MiB |
| `GOMEMLIMIT=32MiB`，4 worker | 1,674,098/s | 2.83 ms | 11.4 MiB | 18.8 MiB |

这些是**独立离线进程的短期实验值**。Go 的 GOMEMLIMIT 是软目标，不限制进程总 RSS；后端的 HTTP、数据库客户端、goroutine 栈、映射内存都可能显著扩大实际占用。**绝对不要直接给正式服务配置 16 MiB**。

## 12–15 秒 CPU 调度对照（GOGC=200，GOMEMLIMIT=32MiB）

| GOMAXPROCS/工作线程 | 吞吐 | p99 | RSS 高水位 |
|---|---:|---:|---:|
| 2 | 1,082,831/s | 1.61 ms | 33.7 MiB |
| 4 | 1,674,098/s | 2.83 ms | 18.8 MiB |
| 8 | 1,399,283/s | 13.68 ms | 42.2 MiB |

宿主 cgroup 配额只有 4 核；本轮 8 worker 没有吞吐优势，尾延迟明显恶化。2 worker 的尾延迟较低，但牺牲吞吐；对多核调度应基于业务负载与优先级做选择，**不建议绑核/NUMA/线程锁定等未经证明的底层改动**。

## 验证、监控盲点与部署建议

- **通过：** Go 1.23.2 独立生产核心模块测试和竞态检查；样本 JSON 逐字节一致性对照；`internal/runtuning` 竞态测试；脚本可重建离线模块并复现负载。报告随包附上原始结构化 JSON，含所有无效和失败的实验设计，而非只保留最佳值。
- **未完成：** 完整后端 `go test ./internal/health` 所需第三方 `pgx/mysql/uuid` 模块未缓存，在 `GOPROXY=off` 下无法完成；未验证数据库、真实设备、通知可靠性、完整服务 HTTP p99、网络队列及 24 小时 GC/RSS 漂移。
- `HeapAlloc` 每 100ms 采样一次，可能漏掉更短暂峰值；`VmHWM` 是进程 RSS 高水位，包含非 Go 堆内存，但不说明其每个内存段。`PauseTotalNs` 是累计 STW 暂停，不包括并发 GC 的全部 CPU 开销。
- 保持 `GOGC=100` 作为默认值；有足够内存和实际业务 A/B 数据时，可显式运行 `GOGC=200`，搭配已存在的 `SUINING_RUNTIME_AUTOTUNE=1`（Go1.23 的 cgroup v2 CPU/内存软目标适配），并监控 `VmRSS/VmHWM`、GC CPU、p99、丢包/重试和告警闭环。Go 环境变量 `GOGC` 原生支持，无须注入未经测试的自动调参代码。
- 对 SOS 和健康数据不可采用基准程序里的非阻塞丢弃模型；生产应采用上游可感知的背压、持久化与明确 ACK，压测中的 drop 仅作为测量失败/过载标记。

## 复现命令（源码包根目录）

```bash
python3 scripts/go123-deep-hardware.py --duration 30s --rates 0 --gogc 100 --workers 4 --output run100.json
python3 scripts/go123-deep-hardware.py --duration 30s --rates 0 --gogc 200 --workers 4 --output run200.json
python3 scripts/go123-deep-hardware.py --duration 9s --rates 3000 --gogc 200 --workers 4 --output paced200.json
GOMEMLIMIT=32MiB python3 scripts/go123-deep-hardware.py --duration 15s --rates 0 --gogc 200 --workers 4 --output limit32.json
```

需要 `go version` 为 1.23.x，Python 3，以及 Linux 才能有 VmHWM；默认 `GOPROXY=off`。每次工具先运行真实核心功能及 `-race` 检查，并在单独进程中运行性能负载。压测结果是经验数据，不能用其为商业照护业务承诺 SLA。
