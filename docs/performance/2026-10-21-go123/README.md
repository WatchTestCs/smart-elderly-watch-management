# Go 1.23 后端实测（2026-10-21）

此目录保留隔离核心测试与 CPU/GC 的**原始性能记录**。采集环境 Go 1.23.2，linux/amd64，4 CPU cgroup 配额；数据并非生产 HTTP/MySQL 吞吐或业务 SLA。

- `expanded-report.md`: 方法、结果、局限、复现命令。采集时原始源码 `go.mod` 仍为 Go 1.24；本次提交另作 Go 1.23 兼容改动，包括 `go.mod` 和基准测试 `B.Loop`。**兼容改动后的完整服务尚未完成依赖编译和数据库集成验证**。
- `coreload-*.json`: 各轮 10 秒 CPU-only 负载和内存 GC 数据（顺序 A/B/B/A）；不含数据库和 HTTP。
- `bench-core.txt`: 独立 CPU 统计聚合对照微基准。
- `test-race.txt`, `query-race.txt`: 仅隔离模块的 Go 1.23 竞态测试日志。

完整后端 `go test ./...`、MySQL 的 500/503 和通知最终送达尚未在本环境验证。部署前必须在隔离的 MySQL 8.4 环境运行完整检查，核对索引、落库与通知状态一致性。

## 错误演示（不作为性能数据）

- [第一版原定 20 分钟 MySQL 压测的错误演示](./FIRST_20MIN_INCORRECT_DEMONSTRATION.md)：保留历史中止、HTTP 500/503 与测试记录不完整的教训，帮助避免误把中止测试写成完整的 20 分钟性能证明。
- [机器可读证据状态](./first_20min_incorrect_evidence.json)：仅为截图与已知记录的追溯索引，**不是原始压测结果**。
