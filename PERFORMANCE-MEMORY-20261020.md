# Go 1.23 内存与 CPU 调度深度打磨

日期：2026-10-20。源码基线：`suining-go123-top-tier-cpu-20261020.zip`，目标：**仅规则与健康数据 CPU/GC/内存路径**，不把数据库、HTTP 或通知吞吐混入结论。

## 已落地

1. **设备级证据缓冲区复用（生产代码）**：`internal/rules/evidence_fast.go` 提供 `appendEvidenceFast(dst, ...)`。已在 `internal/rules/engine.go` 接入每设备专属 `[512]byte` 工作区，各证据在同步 `tx.Exec` 完成后才复用；不跨设备或异步队列共享、不使用池缓存隐私数据。旧 `marshalEvidenceFast` 仍保持独立返回缓冲区的契约。遇到 Unicode、JSON 需转义、非法时间或数值继续调用兼容标准库的回退路径。
2. **cgroup v2 运行时调度（可选、默认关闭）**：`internal/runtuning` 在设置 `SUINING_RUNTIME_AUTOTUNE=1` 且运行于 Linux 时读取 `/sys/fs/cgroup/cpu.max` 和 `memory.max`。仅当未显式设置 `GOMAXPROCS`/`GOMEMLIMIT` 时，按 CPU 配额向上取整但不超过 CPU 数来调整 Go P 数，并将 Go 的内存软限额调整为 cgroup 硬限额的 70%（至少需要 256 MiB 的硬限额才启用）。其他 OS/没有有效配额时保持原样。`cmd/server/main.go` 初始化时调用；`deploy/platform.compose.yml` 提供关闭的环境开关。**不修改 OS 线程亲和性、NUMA 或内核调度器。**
3. **可复现实测矩阵**：`scripts/go123-memory-scheduling.py` 从包中的生产源码构建无第三方依赖的隔离测试环境，验证规则／健康路径，并对 `GOMAXPROCS={1,2,4,8}` × `GOMEMLIMIT={off,128MiB}` 执行两轮并行基准，生成 `memory-scheduling-matrix.json`。矩阵共 48 条原始测量。日志中 `ns/op` 是*聚合吞吐的倒数*，非真实请求 p99 延迟。

## 性能实测

环境：Go 1.23.2，Linux amd64，Intel Xeon Platinum 8573C；当前容器 CPU 配额 4 CPU（`cpu.max` 为 `400000 100000`），cgroup 内存上限 4 GiB。此机器可能存在共享资源争用，所有结果只代表此次运行。

| 基准 | 优化前 | 本轮 | 结果 |
|---|---:|---:|---|
| 单核完整规则评估＋证据编码 | 2,495–2,655 ns/op | 2,053–2,162 ns/op | 加速约 1.15–1.29 倍 |
| **规则路径分配** | **1,632 B、8 allocs/op** | **32 B、1 alloc/op** | 每次分配字节下降约 **98.0%** |
| 4 个 P、GOMEMLIMIT=off 规则并行聚合吞吐（两次中位数） | 约 75 万输入/s | 约 102 万输入/s | 约 +36%，仍需更长时间复测 |
| 4 个 P、GOMEMLIMIT=128MiB 规则并行聚合吞吐（两次中位数） | 约 73 万输入/s | 约 142 万输入/s | 不可归因于内存软限额；短时噪声明显 |
| 4 个 P、健康预处理＋JSON＋哈希，GOMEMLIMIT=off | — | 约 203 万样本/s | 健康路径未做本轮代码改动 |

矩阵中 `GOMAXPROCS=8` **未稳定胜过 4**；增加 P 未必增加真实处理器配额。`GOMEMLIMIT=128MiB` 的短时表现可能因 GC 时间、测次及系统负载而上下波动，**不构成建议将生产环境统一限制为 128MiB** 的依据。内存软限制不会强制限住进程 RSS，更不能替代容器硬配额或 OOM 监测。

### 未采用的试验

曾将健康 JSON 缓冲预留常量从 240 缩为 176：普通样本分配下降，但带纳秒时间、偏移时区、较长标识与小数值的样本发生二次扩容，从单次分配退化成两次。为避免混合负载回退，**生产健康序列化保留原有容量策略**。未采用 `sync.Pool` 缓存敏感证据、强制 GOGC、NUMA 绑核或汇编，因为尚无证据支持整体增益。

## 正确性与验证

- Go 1.23.2 隔离纯计算：规则、健康代码 `go test -race -count=1`、`go vet` 通过。`TestFastEvidenceByteExact` 在 **12,000 组**规则证据输入上新增了对借用缓冲路径的逐字节等价检查；原 10,000 组样本等价测试及原有测试保持通过。
- `internal/runtuning` 的 CPU 配额、内存限额、无效值与管理员显式环境变量优先级测试：`go test -race`、`go vet` 通过。
- **完整项目仍未能跑通 `go test ./...`**：当前环境没有缓存 `pgx/v5`、`mysql`、`uuid` 等依赖且关闭外部模块获取。`engine.go` 数据库路径仅完成格式和源码调用范围审查，**未通过完整依赖编译/真实数据库集成测试**。设备证据缓冲区经 `tx.Exec` 同步消费后的所有权假设，需要在真实后端中验收。
- 不包含端到端 p50/p95/p99、数据库、网络、通知投递或长期驻留 RSS 测试；短时吞吐不可视为生产 SLA。

## 复现与部署

```bash
# 解压源码 ZIP，在项目根目录运行（Python 3 + Go 1.23.x）
python3 scripts/go123-memory-scheduling.py
# 数据库、网络都不参与测试

# Linux cgroup v2 容器，明确 opt-in；管理员的 GOMAXPROCS 和 GOMEMLIMIT 优先
export SUINING_RUNTIME_AUTOTUNE=1
# 或 docker compose 中设置此变量再部署
```

后续值得优先验证：4P/8P 在长稳态及 CPU throttling 下的实测差异、`runtime/metrics` 分配与 GC 时长、堆峰值和内存 RSS、规则设备周期的 p99，以及完整 Go 服务依赖恢复后的集成验证。当前优化是**内存效率和容器可控调度**，不意味着“硬件性能已达到理论极限”。
