# 2026-10-20 Go 1.23 性能资料归档

本目录保留深层性能实验报告和两份原始数据。四份早期 CPU、分配及调度阶段报告已清理，历史内容可通过 Git 记录查看；本次清理不重新测量。这些 CPU、GC 和内存实验不能代替 MySQL、HTTP 或通知的端到端验收。

| 文件 | 内容 |
| --- | --- |
| [memory-scheduling-matrix.json](memory-scheduling-matrix.json) | 内存与调度参数的原始测量矩阵 |
| [DEEP-LOWLEVEL-PERFORMANCE-REPORT.md](DEEP-LOWLEVEL-PERFORMANCE-REPORT.md) | 持续负载、GC、内存峰值与尾延迟实验 |
| [deep-hardware-raw-results.json](deep-hardware-raw-results.json) | 深层性能实验的原始数据 |

复现脚本位于仓库根目录的 `scripts/`：`go123-perf-polish.py`、`go123-memory-scheduling.py`、`go123-deep-hardware.py`。调度矩阵及深层实验的默认输出已指向本目录；脚本运行会写入测量结果，历史数据需要保留时请先备份或使用深层实验的 `--output` 指定其他位置。

[后续性能证据](../2026-10-21-go123/README.md) · [最新双库混合负载报告](../2026-10-22-backend-hybrid/README.md) · [项目首页](../../../README.md)
