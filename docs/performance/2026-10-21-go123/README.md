# Go 1.23 后端实测（2026-10-21）

此目录保留隔离核心测试与 CPU/GC 的**原始性能记录**。采集环境 Go 1.23.2，linux/amd64，4 CPU cgroup 配额；数据并非生产 HTTP/MySQL 吞吐或业务 SLA。

- `expanded-report.md`: 方法、结果、局限、复现命令。采集时原始源码 `go.mod` 仍为 Go 1.24；本次提交另作 Go 1.23 兼容改动，包括 `go.mod` 和基准测试 `B.Loop`。**兼容改动后的完整服务尚未完成依赖编译和数据库集成验证**。
- `coreload-*.json`: 各轮 10 秒 CPU-only 负载和内存 GC 数据（顺序 A/B/B/A）；不含数据库和 HTTP。
- `bench-core.txt`: 独立 CPU 统计聚合对照微基准。
- `test-race.txt`, `query-race.txt`: 仅隔离模块的 Go 1.23 竞态测试日志。

原始离线采集阶段未能验证完整后端。**此后** [Preview CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39623700602) 已在 Go 1.23.2 / 1.27.2 与一次性 MySQL 8.4、PG 16 服务上通过所配置的后端测试，但这与本目录 CPU-only 测试是不同证据来源。真实外部通知送达及生产长期可靠性仍未验证。

## 20 分钟 MySQL 真实负载：失败结果（独立于上面的 CPU-only 基准）

[运行 #39622692722](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39622692722) 在 Go 1.23.2 / MySQL 8.4 的隔离 CI 环境中执行了完整时长的合成健康上报与并行规则评估。**测试用例结果 FAIL；当时工作流被 `go test | tee` 管道错误地标为成功**。检查运行日志和 Artifact，而不是工作流状态徽章。

| 指标 | 原始运行日志确认 |
| --- | ---: |
| 请求尝试 | 1,987 |
| 应用成功确认 | 1,959 |
| HTTP 500 / 503 | 28 / 0 |
| 规则评估错误 | 19 |
| 成功确认样本 | 391,800 |
| `health_sample` / `health_change` | 391,800 / 391,800 |
| 测试用例总执行时长 | 约 1,233 秒（包含测试结束核对） |

该运行提供了重要的**数据一致性线索**，却没通过 20 分钟稳定性门槛。约 17 分钟时开始出现 HTTP 500；原因不能仅凭状态码断定，需根据慢查询、锁和连接池数据追查。该测试使用单应用 `httptest` + 真 MySQL，不涉及公网代理、真实设备及外部通知发送。延迟 p99 等更细数据应读取原始 Artifact，不能从表格反推。

## 错误演示（不作为性能数据）

- [第一版原定 20 分钟 MySQL 压测的错误演示](./FIRST_20MIN_INCORRECT_DEMONSTRATION.md)：保留历史中止、HTTP 500/503 与测试记录不完整的教训，帮助避免误把中止测试写成完整的 20 分钟性能证明。
- [机器可读证据状态](./first_20min_incorrect_evidence.json)：仅为截图与已知记录的追溯索引，**不是原始压测结果**。
