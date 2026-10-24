# 限流中间件分片与环形缓冲（2026-10-24）

基线 `ac1fceb`（security-fixes 分支）。目标：多核高并发下消除限流热路径的锁争用与分配，语义逐字不变。

## 实现（internal/ratelimit/ratelimit.go）

1. **16 分片各持锁**：`[16]shard`，二级 map（IP→路径→记录）替代 `IP+"|"+path` 拼接键，热路径零字符串分配；分片路由用手写 FNV-1a（IP+路径散列，不产生拼接）。
2. **每 key 时间戳环形缓冲**：容量懒增长且严格 ≤ limit；`time.Now()` 移入分片锁内取——同 key 追加即时间顺序（单调钟不回退），head 恒最旧，出窗 O(1) 出队，替代原全量过滤扫描。
3. janitor 循环体抽为可注入时间的 `cleanup`；`limits` 表、`clientIP`、`Middleware`（429+Retry-After=600+响应体）与 `New/Allow/Middleware` 签名逐字未动。

窗口判据 `now.Sub(t) >= window` 与原保留条件 `now.Sub(t) < window` 严格互补；拒绝路径不追加不扩容，单 key 内存上界为 limit 个时间戳（1800 上限 ≈43KB，与原实现同阶）。无 unsafe、无 API 变更。

## 测量

Windows 本机（Ryzen AI 9 H 365，20 逻辑核），Go 1.23.12 引导（按 go.mod toolchain 编译 1.26.10），`-cpu 1,4,8 -count=3` 中位数，RunParallel 吞吐摊销值（非单请求延迟）。原始记录：[before](rl-bench-before.txt)、[after](rl-bench-after.txt)。

| 场景（keys=64，4 核） | 原 ns/op | 新 ns/op | B/op、allocs/op |
| --- | ---: | ---: | --- |
| device/report（上限 1800） | 6289 | **32.8（~192×）** | 32B/1 → **0/0** |
| login（上限 30） | 343.9 | **34.6（~10×）** | 32B/1 → **0/0** |
| 限流表外路径 | 1.89-1.93 | 1.83-2.03（噪声） | 0/0 → 0/0 |

全矩阵（1/64/1024 keys × 1/4/8 核）：限流路径 2.8×~192×，**单项退化：无**（单核亦 32×~103×，不存在硬件先例文档中的单线程权衡）。同 key 在 8 核约 111ns 仍高于跨 key（44ns）：同 key 必须原子计数，分片消除的是跨 key 争用，属语义要求。

## 验证

- 新增 8 个语义回归测试全过：精确限次（第 limit+1 次拒绝）、并发同 key 恰 1800 放行、跨分片独立、窗口边界（9分59秒在窗/满 10 分钟出窗）、janitor 重置、表外/GET 零状态短路、429 响应逐字节断言、环形缓冲回绕+容量上界。
- 双库全量回归（MySQL 8.4 + PostgreSQL 16）其余全部通过；`internal/analytics` 的积压清空测试在整包并行负载下墙钟敏感失败一次（清空速率与通过轮次一致 10.8k rows/s），隔离重跑 3/3 通过——与本次改动无路径交集，属既有负载敏感测试。

## 边界

- 限流仅覆盖 `limits` 表内 POST 路径（14 条）；表外路径前后均锁外短路。
- 每 key 环形缓冲常驻至 janitor 回收；空闲 key 由 janitor 在窗口 tick 后删除。
