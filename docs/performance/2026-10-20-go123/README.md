# 2026-10-20 Go 1.23 性能资料归档

本目录归档原先位于仓库根目录的五份报告和两份原始数据。只调整位置，不重新测量，也不改变历史报告的结论与适用范围；这些 CPU、GC 和内存实验不能代替 MySQL、HTTP 或通知的端到端验收。

| 文件 | 内容 |
| --- | --- |
| [GO123-PERFORMANCE-REPORT.md](GO123-PERFORMANCE-REPORT.md) | Go 1.23 核心优化及吞吐验收边界 |
| [PERFORMANCE-POLISH-REPORT.md](PERFORMANCE-POLISH-REPORT.md) | 规则求值、证据编码与分配优化 |
| [PERFORMANCE-TOP-20261020.md](PERFORMANCE-TOP-20261020.md) | CPU 热点和 JSON 快路径优化 |
| [PERFORMANCE-MEMORY-20261020.md](PERFORMANCE-MEMORY-20261020.md) | 内存与 CPU 调度取舍 |
| [memory-scheduling-matrix.json](memory-scheduling-matrix.json) | 内存与调度参数的原始测量矩阵 |
| [DEEP-LOWLEVEL-PERFORMANCE-REPORT.md](DEEP-LOWLEVEL-PERFORMANCE-REPORT.md) | 持续负载、GC、内存峰值与尾延迟实验 |
| [deep-hardware-raw-results.json](deep-hardware-raw-results.json) | 深层性能实验的原始数据 |

复现脚本仍位于仓库根目录的 `scripts/`：`go123-core-benchmark.py`、`go123-perf-polish.py`、`go123-memory-scheduling.py`、`go123-deep-hardware.py`。调度矩阵及深层实验的默认输出已指向本目录；脚本运行会写入测量结果，历史数据需要保留时请先备份或使用深层实验的 `--output` 指定其他位置。

[后续性能证据](../2026-10-21-go123/README.md) · [最新双库混合负载报告](../2026-10-22-backend-hybrid/README.md) · [项目首页](../../../README.md)
