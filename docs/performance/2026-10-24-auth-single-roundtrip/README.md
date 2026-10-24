# 认证会话每请求往返削减（2026-10-24）

基线 `ac1fceb`（security-fixes 分支）。目标：降低每个已登录请求的数据库往返与延迟，不引入缓存、不改变即时撤销语义。

## 发现

每个携带有效会话的请求此前执行 **4 次 `web_session`/`auth_factor` 往返**：

1. `csrf.Middleware` 调 `store.FromRequest`（POST 校验 X-CSRF-TOKEN / GET 注入上下文）；
2. 该 `Get` 内部先读 `web_session.payload`，解密后再查 `auth_factor.generation`（MFA 校验）——2 次往返；
3. `auth.Middleware` 再次 `store.FromRequest`——又 2 次往返。

## 实现

1. **auth 复用请求内会话**：csrf 中间件已把会话放入 request context；auth 中间件优先 `session.FromContext`，仅上下文为空时回退 `FromRequest`。语义：会话在本请求开始时有效即可；撤销自下一个请求起生效（与逐中间件实时读取的差别仅为同一请求内的微秒级窗口）。
2. **payload 与 generation 合并为一条 SQL**：generation 经 `web_session` 自身的 `role`/`actor_id` 列（与 payload 在 `Create` 中同一事务写入，恒一致）关联子查询取回。`NULL` 对应旧两次查询的 `ErrNoRows` 分支：启用了 MFA 的会话缺行即拒绝。仍是每请求实时读库，多副本即时撤销不变；无跨请求缓存。

净效果：**每已登录请求 4 次往返 → 1 次**；硬件端点与匿名请求不变。

## 测量

Windows 本机 Docker MySQL 8.4（loopback TCP），Go 1.26.10，`GOMAXPROCS=20`，`-benchtime=200x -count=5`。基准 `internal/session/database_bench_test.go`：已启用 MFA 的 ADMIN 会话完整 `Get`（idHash + 读库 + AEAD 解密 + 反序列化 + generation 比对）。A/B 用同一基准文件：候选为工作区，基线经 `git stash` 还原 session.go/auth.go 后运行。

| 实现 | ns/op（5 轮） | 中位 | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| 基线（两次往返） | 791074 / 963567 / 785182 / 858356 / 839012 | ~839012 | ~5050 | 71 |
| 候选（单次往返） | 531762 / 434262 / 466434 / 433714 / 465184 | ~465184 | ~4600 | 53 |

单次 `Get` 中位耗时 -45%（约省一个 loopback 往返 ~374µs），分配 -18 次。中间件层的收益按每请求 3 次往返叠加；loopback 约为每请求 -1.1ms 量级，跨网部署随 RTT 放大。共享宿主容器先后执行，无统计显著性保证；数值不可换算为整体吞吐倍数。

## 验证

- `go test ./internal/session ./internal/auth` 通过。
- 真实 MySQL 8.4：`./internal/session ./internal/db/... ./cmd/server` 集成测试通过（含数据库会话共享、撤销、factor generation 吊销用例）。
- `internal/db/dbtest` 的 `MySQL(t)` 放宽为 `testing.TB` 以支持集成基准，对现有 `*testing.T` 调用方源兼容。

## 边界

- 不改变：会话 TTL、登录换发、撤销 SQL、MFA 语义、内存后端行为。
- `web_session.role/actor_id` 列成为 generation 匹配的依据（此前为 payload 解密值）；两者由同一 INSERT 原子写入，一致性由 `Create` 保证。
