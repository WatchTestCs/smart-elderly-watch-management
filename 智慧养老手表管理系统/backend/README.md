# 岁宁 Go 后端 · 2.1.0

Go 服务负责业务 API、设备接入、会话、图片、内嵌前端、照护与通知作业，以及分析同步。**MySQL 是唯一业务主库，PostgreSQL 是异步分析库**；Windows 由 Tauri 管理 Go 生命周期，也可独立部署服务用于受控设备接入和长期值守。

[整体架构](../../README.md#系统架构) · [双库部署与迁移](../../docs/DUAL_DATABASES.md) · [照护部署](../../docs/CARE_SERVICE_DEPLOYMENT.md) · [源码 Release 与开发分支区别](../../docs/DOWNLOADS.md)

## 技术与模块

Go 模块名为 `suining-care`，声明最低 **Go 1.26.0**、工具链 Go 1.26.10；CI 覆盖 Go 1.26.10 与 1.27.2。HTTP 使用标准库 `net/http`，主库存取使用 `database/sql` 与 `go-sql-driver/mysql`，PG 投影及旧库工具使用 `pgx/v5`。当前没有独立消息中间件或分布式事务协调器。

| 位置 | 职责 |
| --- | --- |
| `cmd/server` | HTTP、后台连接恢复、迁移发布、桌面就绪与父进程监测 |
| `internal/handlers` | 账号、人员、设备、健康、公告、照护与诊断 API；图片与静态资源 |
| `internal/auth` / `session` / `csrf` / `ratelimit` | 角色、BCrypt、会话（默认数据库后端，可选单实例内存模式）、CSRF 与限流 |
| `internal/care` | 通知投递、超时升级与照护后台作业 |
| `internal/db` | MySQL 连接与 SQL 适配、事务、迁移、旧 PG 兼容 |
| `internal/analytics` | PG schema、outbox 同步、事件去重、墓碑与回填 |
| `cmd/migrate` | 默认执行 MySQL 迁移；显式参数支持旧 PG 迁移 |
| `cmd/transfer-postgres` | 旧 PG 到空 MySQL 的离线预检查与事务导入 |
| `cmd/analytics-backfill` | 将当前 MySQL 数据加入分析回填队列 |
| `migrations` / `VERSION` | 编译内嵌 SQL 与统一产品版本 |

SQL 适配覆盖本应用使用的语法，不是任意 PostgreSQL SQL 的通用翻译层。应用服务器显式打开 MySQL，主库故障不会自动切到 PG。

## 运行配置

MySQL 参数默认采用驱动插值传输，显式 DSN `interpolateParams=false` 可回退二进制 prepare。只由驱动编码参数，不改变业务事务和持久性。JSON 参数使用 `json.RawMessage`，BLOB 保持 `[]byte`；完整边界与回退见双库文档。

在本目录启动，替换开发环境占位密码：

```powershell
$env:MYSQL_DSN='suining:替换密码@tcp(127.0.0.1:13306)/suining'
$env:PG_ANALYTICS_URL='postgres://suining_analytics:替换密码@127.0.0.1:15432/suining_analytics?sslmode=disable'
$env:ADDR='127.0.0.1:8795'
$env:DATA_ENCRYPTION_KEY_FILE='替换为已备份的持久化密钥文件路径'
go run ./cmd/server
```

| 变量 | 含义 |
| --- | --- |
| `MYSQL_DSN` | 必须指定库名的 MySQL 原生 DSN，无默认凭据 |
| `MYSQL_MAX_IDLE_CONNS` | 主库应用池空闲连接上限，默认 20、允许 0–20；连接预算紧张可设 5 恢复旧行为 |
| `HEALTH_LATE_DEVICE_LOCK` | **实验性**健康写入锁顺序，默认关闭；仅全体旧写入实例停机排空后可统一设为 1，禁止滚动升级混用 |
| `PG_ANALYTICS_URL` | 独立 PG 分析库 URL；不配置仍可运行核心业务 |
| `ADDR` | 直接启动默认 `127.0.0.1:8080`；前端开发代理使用 8795 |
| `IMAGE_DIR` | 直接启动默认 `./data/images` |
| `SESSION_TTL_HOURS` | 会话期限，默认 12 小时 |
| `SESSION_BACKEND` | 默认 database；memory 仅限显式选择的单实例开发 |
| `HEALTH_HOT_CACHE_MIB` | 健康统计内存缓存预算，默认 16 MiB；0 禁用，允许 0–256；无效值回退 16 |
| `DATA_ENCRYPTION_KEY` / `DATA_ENCRYPTION_KEY_FILE` | 必须提供 base64 编码的 32 字节随机密钥；所有实例共享，不得每次启动更换 |
| `METRICS_TOKEN` / `METRICS_TOKEN_FILE` | `/metrics` Bearer 凭据，至少 24 字符 |
| `DESKTOP_TOKEN` | 初始化与桌面退出的随机令牌，不交给网页 |

桌面由 Rust 将 `SUINING_MYSQL_DSN` / `SUINING_PG_ANALYTICS_URL` 转交给 Go，并传入回环动态端口、本地目录、随机令牌和父 PID。旧 `DATABASE_URL` / `SUINING_DATABASE_URL` 不作为业务主库配置。示例只用于受控本机；远程连接使用验证证书的 TLS，不提交真实凭据。

桌面启动前也需在父进程环境中设置 `DATA_ENCRYPTION_KEY_FILE`，由后端继承；未配置时数据库 API 不开放。仓库根目录的 `python scripts/start-platform.py --validate` 可一次生成开发部署密钥；不要提交生成的 secrets。升级、历史备注回填、DDL 回滚与具体加密范围见 [平台改造说明](../../docs/PLATFORM_UPGRADE_2.0.6.md)。

前端 `npm run build` 的结果复制到 `internal/handlers/web` 后由 Go 编译内嵌。源码归档只保留占位资源；纯 API 开发可以直接运行，桌面发布需执行 [完整构建](../README.md#windows-完整构建)。

## 热数据内存缓存与磁盘历史

`POST /api/v1/health/statistics` 对重复统计结果使用进程内 LRU 缓存，最多 2,048 项、每项最多 64 KiB、入缓存后 30 秒过期。缓存按 JSON 字节数加每项 256 字节固定开销计费；这是缓存预算，不是整个 Go 进程的堆或 RSS 硬上限。完整请求哈希为固定长度键，包含老人、来源、设备、指标、语义和带报告偏移的时间范围，避免长请求字符串常驻内存。

每次查询仍先鉴权，并读取 MySQL 已提交的老人修订号。任何实例通过 `health.SaveBatch` 写入、更正或删除样本后，旧结果即不能命中；未命中计算前后修订号不同则不入缓存。修订号校验失败回到数据库查询，不返回未经核验的缓存。直接绕过样本写入协议修改数据库时，必须同步更新修订号；维护导入、恢复备份后应重启应用。

所有采样及修订历史仍在 MySQL 中先事务提交，再确认写入；PG 分析投影继续异步持久化。进程重启或缓存淘汰后，统计从数据库重建。这里的“冷”表示留在持久化存储，数据库仍按其缓冲池策略缓存经常访问的页，没有强制把历史页移出内存。告警基线和每日统计的 pending/ready 状态继续走原路径，保留随时间变化的计算语义。

此缓存加速同一修订号下的重复统计；持续上传会降低命中率，不能用命中吞吐代表采样落库性能。可通过 `/metrics` 的 `suining_health_cache_hits_total`、`suining_health_cache_misses_total`、`suining_health_cache_validation_errors_total`、`suining_health_statistics_db_loads_total` 观察效果。设置 `HEALTH_HOT_CACHE_MIB=0` 并重启即可恢复直接数据库查询，无新增表、迁移或服务依赖。

## 健康写入设备锁与绑定一致性（实验性）

默认的 `healthBatch` 沿用 **device 行锁 → health_owner_clock 行锁 → 样本与历史事务提交** 顺序。在单设备突发和规则调度竞争时，设备心跳可能等待健康写入完成。本项目提供 **默认关闭** 的 `HEALTH_LATE_DEVICE_LOCK=1` 可选路径：先基于已鉴权的设备绑定写入未提交的健康样本与不可变修订历史，在提交前再获取设备行锁，并重新验证设备号、密钥、启停状态、老人 ID 和绑定 epoch；若绑定已变化，整笔事务回滚并返回冲突，绝不把旧绑定数据确认给新老人。

**切换安全前提：** 新路径的加锁顺序是 **health_owner_clock → device**，与旧路径相反。绝不能让旧写入实例与新写入实例同时处理健康上报；否则可能出现 InnoDB 死锁、500 错误或可用性下降。启用前必须停止所有旧实例与消息写入器、等待既有事务完成、按同一配置启动所有实例，再逐步恢复接入。回退同样先排空所有新路径事务、统一将环境变量取消或设 0，再恢复旧服务。仅凭环境变量热切换或普通滚动更新不满足安全条件。此功能没有数据库迁移，并不允许降低事务持久性或自动重试未知提交结果。

[MySQL 8.4 设备行锁/修订号锁隔离实验](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39763591263) 显示，在人为阻塞同一老人修订号的情况下，原顺序让同设备心跳在 200 ms 观察窗口内不能完成；延后设备锁的试验在约 2 ms 内完成心跳，健康样本与变化历史均完整提交。解绑与写入并发时，新路径检测 epoch 变化并回滚。这不是完整 20 分钟 HTTP 压测的 p99 结论，多实例锁顺序交错仍需进一步验收；因此默认不启用。

## Go 1.23 内存与容器 CPU 调度（可选）

- 规则证据 JSON 现在按**设备处理调用**复用一个私有 512 B 缓冲区；每次 `tx.Exec` 同步返回后才允许复用，异步消费者不得持有该字节切片。序列化失败或需要兼容编码时仍回退标准库。
- Linux **cgroup v2** 下可以设置 `SUINING_RUNTIME_AUTOTUNE=1`，在进程启动时读取 `cpu.max` 和 `memory.max`：仅当管理员没有设置 `GOMAXPROCS` 或 `GOMEMLIMIT` 时，分别设置并发 Go P 上限与保留约 30% 余量的 Go 内存**软限制**。默认关闭；Windows/非 cgroup v2 不改变设置。不要把内存软限额当成防 OOM 保证，也不会设置 CPU 亲和性或线程绑核。
- 容器 CPU 配额会限制可用时间片，Go 1.23 的默认 `GOMAXPROCS` 可能未考虑配额；线程数过高会增加争用和调度开销。用户显式 `GOMAXPROCS=4` 等值始终优先。没有通过性能验证前，不自动修改 `GOGC`、NUMA 亲和性、CPU affinity 或使用 `runtime.LockOSThread`。
- 可在项目根目录执行 `python3 scripts/go123-memory-scheduling.py`：运行独立、离线、标准库规则／健康代码的正确性检查，并对 GOMAXPROCS=1、2、4、8 与 GOMEMLIMIT=off/128MiB 做对照。输出 `docs/performance/2026-10-20-go123/memory-scheduling-matrix.json`。数字**不包含 MySQL/HTTP/通知，不能代表服务端吞吐或延迟 SLA**。

## 事务与分析数据流

业务处理器在 MySQL 中完成建档、绑定、SOS、处置和复核。MySQL 触发器将有关健康、活动、轨迹及照护的分析事件与业务变更一同提交；回滚不会泄露分析事件。Go 分析作业写入 PG 后才确认队列，重复投递按事件号去重，删除墓碑防止旧事件复活。

PG 保存 `suining_analytics.record` 投影，并提供 `daily_health`、`care_summary` 视图；业务页面仍读取 MySQL 当前数据。PG 未配置或断线时队列继续保留，恢复后自动补齐，需监控主库容量。分析投影不能作为主库热备或自动切换目标。

照护任务支持接单、开始处置、记录措施、另一名主管独立复核，状态与履历由事务维护；复核关闭更新对应告警。护理员／审计员岗位限制覆盖照护功能，其他后台权限保持现状。通知发送、送达回执与人员到场分别确认。

## 迁移与存量数据

服务启动自动迁移 MySQL，也可运行 `go run ./cmd/migrate`。SQL 编入程序，不依赖当前工作目录。MySQL 使用连接级锁和脚本校验和；DDL 会隐式提交，采用可重复语句和最终版本记录，不承诺整份 DDL 事务回滚。PG 分析迁移使用事务和 advisory lock，仅维护分析 schema。

本次修复包含 `0008_rule_baseline_source.sql` 的来源前缀索引，以及 `0009_rule_baseline_cover.sql` 的虚拟列和覆盖索引。常见来源设备标识的基线聚合从索引读取数值，继续核对完整二进制来源；较长标识保留原路径。升级前备份并预留索引构建的磁盘空间与维护时间，不能承诺在线升级零中断。`0007` 修订号迁移要求停止旧写入器；不能让旧版写入器与新协议混跑。仅启停本次统计缓存不需要额外迁移。

旧业务 PG 导入前备份旧库与图片，停止新旧服务写入，配置 `LEGACY_POSTGRES_URL` 和目标 `MYSQL_DSN`：

```powershell
go run ./cmd/transfer-postgres
go run ./cmd/transfer-postgres --apply
```

工具支持 PG 0001–0006，以及含健康采样的 0007／0008，默认只预检查；应用导入使用只读源快照和目标单事务，保留 ID、哈希与 NULL，逐表核对行数。目标非空拒绝导入；失败回滚目标业务及分析事件，建表阶段独立。图片另行复制，历史错数据需另行核对，Java/H2 不直接支持。

`go run ./cmd/migrate --legacy-postgres` 仅用于旧 PG 结构维护，显式读取 `DATABASE_URL`，不切换当前服务器主库。新 PG 分析库准备好、修改配置并重启后，可运行 `go run ./cmd/analytics-backfill --apply` 回填。操作权限与切换核对见 [双库指南](../../docs/DUAL_DATABASES.md)。

## 本地回归

```powershell
go test -count=1 ./...
go vet ./...
# 在独立测试实例上配置真实 DSN；不要指向生产环境
$env:TEST_MYSQL_DSN='测试MySQL原生DSN'
$env:TEST_DATABASE_URL='测试PG连接URL'
go test -race -count=1 -timeout 240s ./...
```

MySQL 测试创建、清理随机测试数据库，需要相应权限；PG 测试使用隔离 schema。缺少对应变量时集成测试会跳过，不能称为集成通过。浏览器模拟 API 无法替代真实数据库测试。已有开发阶段的双库、迁移和故障恢复结果见 [2.0.6 说明](../../docs/releases/2.0.6.md)，此次健康采样整合的运行验证另见 [整合记录](../../docs/INTEGRATION_2.0.6.md)。

## 2026-10-21–22 后端性能与风险复核

- [最新修复、双库混合压测与冷热缓存报告](../../docs/performance/2026-10-22-backend-hybrid/README.md)：旧 `b6aba05` 基线加修复完成本地 20 分钟负载，9,484 批、1,896,800 条采样，HTTP 500/503 均为 0，整机平均 CPU 98.41%；分析队列仍有 833 条积压，写入 p99 约 14.8 秒。
- 同报告中的重复统计基准为 23.252 ms → 0.089078 ms，约 261 倍，只针对同一修订号的缓存命中；数据库已经预热，不是物理冷盘或采样上传容量。当前与更新 `preview` 合并后的源码另作定向验证，**未重新执行 20 分钟压测**，历史未全绿的整套测试限制继续保留。

- [Go 1.23 / 1.27 双版本 CI 验证](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39623700602)：使用独立的 MySQL 8.4、PG 16 CI 服务执行后端测试，结果通过。
- [20 分钟真实 MySQL 合成负载](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39622692722)：**实际失败**；1,987 次请求、28 次 HTTP 500、19 次规则错误，391,800 条 ACK 样本及两张持久化表行数一致。不要仅凭工作流绿色状态判断通过；旧脚本漏用了 `pipefail`。
- [第二轮 MySQL 耐久复测](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39629502257)：修正 `pipefail` 后 CI **正确失败**；记录 44 次请求错误、19 次规则错误，387,200 条 ACK 样本与两张数据库表的行数一致。该轮未满足 20 分钟零错误验收，后续本地修复版结果另列于上面的最新报告。
- [原定 20 分钟但中止的错误演示](../../docs/performance/2026-10-21-go123/FIRST_20MIN_INCORRECT_DEMONSTRATION.md)：历史截图无完整原始证据，不能算作已完成长压测。
- [微基准和证据索引](../../docs/performance/2026-10-21-go123/README.md)、[连接池与 SQL 排查](../../docs/MYSQL_503_500_CONTINUATION_20261021.md)：区分模拟负载、数据库集成、完整 HTTP 路径和生产指标。

不能把单实例 `httptest` + MySQL、独立 CPU 微基准或模拟通知网关结果推广成真实机构 SLA。生产上线仍需实际设备、可靠通知回执、人员接单、故障恢复与权限隐私验收。

## 故障与安全边界

- HTTP、静态页面和桌面就绪不等待数据库；MySQL 迁移成功后才发布业务 API，失败后后台重试。
- `GET /healthz` 只代表进程存活；`GET /api/v1/service/status` 返回主库就绪及分析配置、连接状态，不暴露连接串。MySQL 未就绪时业务 API 返回 `503 DATABASE_UNAVAILABLE`；探测前正在执行的 SQL 也可能先失败。
- MySQL 业务池仍为最多 20 个连接，真实就绪探测使用另一个最多 1 连接的池，部署应为每个应用副本预留探测额度。保留已有探测和业务池指标；离线设备维护每页最多 100 行，跳过忙设备，在后续周期重查。
- MySQL 恢复后检查会话；失败业务写入不自动重放。分析队列只重试已经提交的分析事件。
- 首位管理员必须经桌面令牌初始化，无默认账号；普通写接口校验会话、角色和 CSRF，设备接口单独鉴权。会话默认存于数据库；显式启用 `SESSION_BACKEND=memory` 时才采用单实例内存会话。
- 桌面本地服务随窗口退出。长期值守需常驻部署、通知网关和人工流程；图片需与 MySQL 分别备份。

### Go 1.23 持续运行 GC、p99 与内存调度基准（CPU-only）

- `python3 scripts/go123-deep-hardware.py --duration 30s --rates 3000 --gogc 100 --output gc100.json`
- `python3 scripts/go123-deep-hardware.py --duration 30s --rates 3000 --gogc 200 --output gc200.json`
- `--rates 0` 为无排队、闭环饱和 CPU 基准；正数为每秒计划批次数，每批 200 样本，包含标准化、真实快速 JSON 编码和身份 SHA-256。仅使用 Go 1.23 标准库，在独立隔离模块中复制**本仓库生产代码**，不包含 HTTP、数据库、规则持久化、通知与磁盘写入。
- 结果独立记录 **实际接受、队列拒绝、发包器漏节拍**；发包漏节拍超过 2% 或发生队列拒绝时标记 `pacing_valid=false`，不得以该次 `p99_us` 同正常试验直接对比。p99 来自 10 μs 分桶，超过 200 ms 的尾部落入溢出桶。
- 工具每 100 ms 采样 Go HeapAlloc（可能漏掉更短峰值），并读取 Linux `/proc/self/status` 的 VmHWM 作为进程最大 RSS；`GCCount`、`GCPauseMs` 来自 Go 运行时累计计数，两者均不代表完整 GC CPU 消耗。
- `GOGC=200` 可减少高分配工作负载的 GC 频率，但会增加活跃堆/驻留内存；**不自动开启**。在有余量的容器中可显式设置 `GOGC=200`，并联合监控 RSS、内存软限额、p99、队列等待、失败与真实消息持久化。小内存部署不能盲目提高 GOGC。
- 禁止用标准库 CPU 基准声明生产系统 SLA，禁止在高压下静默丢弃健康数据或 SOS；应用层只能显式背压/拒绝并让上游确认重试。

MySQL 业务连接默认每条物理连接缓存最多 64 个预编译语句，减少重复 SQL 解析与大参数转义；20 条连接默认最多 1280 个服务器句柄/副本，须计入服务器语句预算。`MYSQL_STMT_CACHE_SIZE=0` 恢复插值/临时 prepare 路径；可设置 0..128。冷启动与 SQL 形状抖动可能变慢，最终延迟对照见 [PR #30](https://github.com/WatchTestCs/smart-elderly-watch-management/pull/30)。

业务 MySQL 连接默认 READ COMMITTED。已知 session 默认未被修改时，业务事务复用该隔离设置，减少一次往返；未知 SQL/显式会话变更后恢复每事务设置。裸 SQL() 工具如需其他隔离须显式指定。This also sets the default isolation of raw SQL() connections; tools requiring another isolation level must request it explicitly.

## API 入口（破坏性变更）

Go 原生路由直接公开 `/api/v1/`，拒绝 `/api/v1/`，不提供别名回退。`SUINING_SESSION` 是仅 HTTP 使用的服务端会话 Cookie；客户端必须按 `/api/v1/auth/session` 获取 CSRF、使用新 Cookie 重新登录。接口清单由 `scripts/generate-openapi.py` 生成，见 `internal/apidoc/openapi.json`。MySQL 主库、可选 PostgreSQL 分析副库职责不变。
