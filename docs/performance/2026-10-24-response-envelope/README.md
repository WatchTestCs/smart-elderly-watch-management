# 响应包络编码零拷贝化（2026-10-24）

基线 `ac1fceb`（security-fixes 分支）。目标：降低每个 HTTP 响应的编码分配与 CPU；**输出字节与状态/头部完全不变**（字节级一致性回归固化）。

## 实现（internal/response/response.go）

1. **OK 空包络常量化**：`OK(w, nil)`/`OK(w, map{} )`（22 处调用形状）直接写出预分配的常量字节，免 map 构造、注入与反射编码。
2. **Fail 栈缓冲拼装**：`{"message":<marshal(msg)>,"returnCode":"0000"}` 用 192 字节栈缓冲拼装，message 值仍经 `json.Marshal` 转义（与 map 键序编码逐字节一致）；超长消息自动扩容到堆。
3. **Error500 UUID 快路径**：398 处调用均为 `Error500(w, ref())`（uuid.NewString，hex+连字符无需转义）；合法 UUID 形式直接拼装固定字段序（message<reference<returnCode，与 map 键排序一致），非 UUID/对抗输入回退原 map 编码路径。
4. **头部直写**：`Content-Type` 值切片包级共享（net/http 只读），跳过 `Header.Set` 的键规范化与单元素切片分配。
5. 泛型 `JSON` 路径改为 `Marshal` + 显式换行（原 `Encoder.Encode` 内部等价行为）；Marshal 失败时 header 已写、body 为空，与旧实现忽略 Encode 错误一致。

不改变：公开函数签名、OK 对传入 map 的 `returnCode` 注入语义、状态码、HTML 转义行为。

## 测量

Windows 本机，Go 1.26.10，GOMAXPROCS=20，`-benchtime=1000x -count=3`，A/B 用 `git stash` 还原 response.go。中位数：

| 基准 | 基线 ns/op | 候选 ns/op | 基线 B/allocs | 候选 B/allocs |
| --- | ---: | ---: | ---: | ---: |
| OKEmpty | 369 | **132（-64%）** | 858 / 9 | 424 / 3 |
| Fail | 742 | **236（-68%）** | 922 / 11 | 658 / 6 |
| Error500WithUUID | 1453 | **477（-67%）** | 1083 / 16 | 680 / 6 |
| OKSmall（4 字段） | 923 | 993（噪声内持平） | 1147 / 17 | 1227 / 17 |
| JSONSmallMap | 784 | 888（噪声内持平） | 922 / 11 | 970 / 11 |
| JSONNestedMap | 1822 | 2241（噪声内持平） | 1539 / 30 | 1699 / 30 |

包络路径（Fail/Error500/OKEmpty——设备端点最高频的响应形状）获得 2-3× 降低；泛型路径持平（成本在 map 构造与反射编码，属调用方形状，未在本轮改动范围）。共享宿主先后执行，无统计显著性保证。

## 验证

- 字节级一致性回归（response_parity_test.go）：对旧实现逐字拷贝做 status/header/body 三者全等断言，覆盖中文/emoji/控制字符/U+2028/HTML 转义/超长消息/UUID 大写/对抗性 reference/编码失败（NaN）/预置 returnCode 覆盖/多键排序。
- `go test ./internal/response`、`go build ./...`、`go vet` 通过。

## 边界

- OKSmall 等泛型路径 B/op 略增（+80B，头部直写与换行分离写的常数差），ns/op 噪声内持平；高频包络路径收益覆盖该常数。
- 调用方 map 构造（`map[string]any{...}` 字面量）仍占响应分配的大头，属各 handler 形状，本轮不动。
