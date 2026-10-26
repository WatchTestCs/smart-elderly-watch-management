# 滥用控制与独立监控

## 限流边界

进程内 32 shard，每请求只锁所属 shard；全进程默认总计 8192 key、262144 时间戳槽，配置上限
100 万 key/1600 万槽不是推荐默认值。固定端点集合、规范 IP、有限字符串长度共同约束保留内存。
每个 shard 按预算拒绝新增/扩容，不清除活跃配额；最多检查 16 项过期 entry，后台每分钟回收。
配额仍为精确滑动窗口。高基数攻击可能占满额度并拒绝新正常来源，因此需边缘控制与容量监控。

时间戳在 amd64 的 time.Time 为 24 字节，默认最多约 6 MiB；加上有界 map/window/IP/path 和
分配器开销，保留内存量级由 K、S 上限决定，为 O(K+S)，不会随新增来源无限增长。
Go runtime、在途 HTTP 请求、请求 header、其他缓存不属于这项 limiter 预算，不能把整个应用
RSS 宣称固定 6 MiB。需在反代限制请求/并发，在运行容器设置合理内存预算并监测高水位。
实际微压测证据见 limiter-memory-bf33177：三轮共 30 万模拟来源、IPv4/IPv6、256 goroutine，
key 保持 8192；观测 RSS 峰值约 23.09 MiB。单次观测不是生产内存上限或吞吐承诺。
NAT 共享、高频单 IP、伪造 header、trusted proxy、关闭生命周期与容量拒绝另有 race 回归测试。

TRUSTED_PROXY_CIDRS 最多 64 项，只有 TCP peer 在其中才解析 Forwarded/XFF，最多 4096 字节
和 32 hops，从右向左跳过可信代理。无效链回退 peer，mapped IPv4 规范化、拒绝 zone；直接
客户端伪造 XFF 无效。Nginx 覆盖 XFF 并清除 Forwarded，默认独立网段 172.30.73/24，应用仅
信任 gateway /32。网段冲突时同时调整 PLATFORM_SUBNET、GATEWAY_IPV4、TRUSTED_PROXY_CIDRS。
不能用整个办公网/0.0.0.0/0 当可信代理，也不能让客户端绕过 ingress 直连公开后端。

Nginx shared zones 在同一 ingress 内跨 worker/应用副本统一控制登录与硬件洪水，内存区固定。
多 ingress/多宿主需要上游统一网关/WAF 的共享配额，当前进程 limiter 仍不是分布式总配额。
登录失败审计使用主库 shared 账号+IP 与账号级窗口，两实例可见，防止轮换 IP 分摊撞库；
只有登录写审计，不为每条普通请求增加共享限流 MySQL 写入。并发在途尝试仍需要边缘并发控制。

## 监控链

应用 → Prometheus → Alertmanager → 独立 webhook 接收端。Compose 的 paging profile 读取
oncall-webhook-url/oncall-token 私有文件；未配置不意味着已接入外部值班渠道。
业务启动依赖 MySQL，PG 与监控互不阻塞；/healthz 只是进程活性。

Prometheus 之外运行 `scripts/reliability/watchdog.py`，应由另一宿主/外部 supervisor 管理。
配置 WATCHDOG_STATE（持久 SQLite）、WATCHDOG_PROMETHEUS_URL、WATCHDOG_ALERTMANAGER_URL、
ONCALL_WEBHOOK_URL_FILE、ONCALL_TOKEN_FILE。生产默认 15 秒两次失败，接收端 HTTPS、禁止
redirect；loopback HTTP 仅用于本地演练。token 至少 24 字符。
失败告警保留重试，进程重启仍可续发；同故障 resolved 不越过尚未交付的 firing。测试真实
503 接收端后重启 spool 并核对顺序。接收端 2xx 只记录 receiver_accepted，不记录人工响应。

数据库离线、app 死亡、worker 心跳陈旧、queue 长期未完成、queue 采样失败、TLS 配置无效、
网关 DNS/timeout/network/HTTP 反复失败、可选 PG 故障分别有指标/规则。
本机演练使用正式规则的原始 for 时长，只将抓取/评估/分组间隔缩短为本地等待；不会手动把
任务改为失败、关闭审计、删除失败记录或以 healthz 200 代替 SOS 可用。

同机进程独立只能证明 app/Prometheus 退出时其他进程仍运行，不能证明整机停电/断网可见。
生产必须在异地实际停主机、断网络、验证接收端送达及值班人员接手，并演练备用救援流程。
