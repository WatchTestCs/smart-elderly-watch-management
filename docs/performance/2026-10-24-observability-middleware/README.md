# 观测中间件残余分配清除（2026-10-24）

基线 `cd22f51`（security-fixes 分支）。承接 2026-10-23-hot-path（8777ns/87allocs → 1411ns/28allocs），目标：清除 HTTP 中间件残余分配，日志/指标语义不变。

## 实现（internal/observability/metrics.go）

1. **标签驻留（intern）**：`route=...,method=...,status="..."` 标签串按 (route, method, status) 复合键驻留于 `labelStrings`，与直方图更新同一锁段内查填——重复组合零字符串分配。渲染函数 `buildHTTPLabels` 用 AppendQuote/AppendInt 分段拼装，与旧 `Quote` 拼接逐字节一致。
2. **in-flight 原子化**：`suining_http_in_flight` 从每请求两次全局锁 map 写改为原子计数；idle→busy 迁移时置 once 标记，暴露端按"曾启动即输出（空闲为 0）"复刻旧 map 条目永不消失的行为，外部 `Add` 值仍叠加。
3. **resolver 原子指针**：`SetRouteResolver` 发布完整构建的指针，请求路径免锁读取（原为每请求锁一次读不可变函数值）。
4. **X-Request-Id 直写**：`Header.Set("X-Request-ID",...)` 本就规范化为 `X-Request-Id`——直接用规范键写入跳过每请求两次键规范化分配，单值时原地覆写跳过新切片。
5. **slog.LogAttrs**：替代 `slog.Info(...any)` 的装箱与 argsToattrSlice；键序与值类型（string/int64/float64）一致，输出逐字节相同。
6. **uuid.EnableRandPool()**：包 init 批量化 crypto/rand（256 字节读产 16 个 v4），uuid 值语义不变。

## 测量

Windows 本机（20 逻辑核），Go 1.26.10，`-benchtime=2000x -count=3` 中位数，A/B 用 `git stash` 还原 metrics.go：

| 基准 | 基线 | 候选 | 变化 |
| --- | ---: | ---: | --- |
| RegistryHTTP（完整中间件） | 1639 ns / 24 allocs / 1315 B | **982 ns / 10 allocs / 1102 B** | **-40% ns，-14 allocs** |
| RegistryObserveHTTP | 91 ns / 0 allocs | 92 ns / 0 allocs | 持平 |
| RegistryScrape（100 组标签） | 617817 ns | 574598 ns | 噪声内 |

对照 2026-10-23-hot-path 的历史数据（同基准 8777→1411ns）：本中间件两轮累计 8777 → 982 ns（~8.9×），87 → 10 allocs。

## 验证

`go test ./internal/observability` 全绿（并发快照一致性、慢客户端隔离、桶边界既有测试不弱化）；`go build ./...`、`go vet` 通过。in-flight 暴露行为经 once 标记与旧实现逐字节对齐。

## 边界

- 残余 10 次分配：uuid 生成（已批量化）、writer 包装、slog handler 内部缓冲——进一步消除需自定义 handler 或对象池，收益递减且增加日志路径复杂度，未做。
- `labelStrings` 与 `httpHistograms` 基数相同（有界于路由×方法×状态组合）。
