# 岁宁 Go 后端 · 2.1.0

Go 服务端负责处理业务 API、手表设备接入、服务端会话、图片存储、内嵌前端静态资源、照护通知作业以及分析数据同步。系统以 MySQL 作为唯一业务主库，以 PostgreSQL 作为异步分析库。在 Windows 桌面端中，Go 进程由 Tauri 托管生命周期；在需要全天候接收设备上报的场景下，也可作为独立后台服务常驻运行。

[整体架构](../../README.md#系统架构) · [双库部署与迁移](../../docs/DUAL_DATABASES.md) · [照护服务部署](../../docs/CARE_SERVICE_DEPLOYMENT.md) · [源码与构建说明](../../README.md#源码与构建)

## 技术栈与代码结构

Go 模块名为 `suining-care`，`go.mod` 声明最低版本为 Go 1.26.0（工具链 Go 1.26.10，CI 覆盖 Go 1.26.10 与 1.27.2）。HTTP 层基于标准库 `net/http`，MySQL 访问使用 `database/sql` 与 `go-sql-driver/mysql`，PostgreSQL 分析投影与旧库迁移工具使用 `pgx/v5`，不依赖外部消息队列或分布式事务协调器。

| 目录 / 入口 | 职责说明 |
| --- | --- |
| `cmd/server` | HTTP 服务启动、后台数据库重连、自动迁移、桌面端就绪握手与父进程存活监测 |
| `internal/handlers` | 账号、档案、设备、健康、公告、照护与诊断 API，以及图片读写和内嵌静态资源服务 |
| `internal/auth` / `session` / `csrf` / `ratelimit` | 角色权限校验、BCrypt 密码哈希、会话管理（默认存数据库，可选单实例内存模式）、CSRF 防护与请求限流 |
| `internal/care` | 告警通知投递、超时升级与照护工单后台作业 |
| `internal/db` | MySQL 连接池、SQL 方言适配、事务封装、自动迁移与旧版 PG 兼容层 |
| `internal/analytics` | PostgreSQL 分析表结构、outbox 异步同步、事件去重、删除墓碑与历史数据回填 |
| `cmd/migrate` | 数据库迁移命令行工具，默认迁移 MySQL，可通过参数指定迁移 PG |
| `cmd/transfer-postgres` | 旧版 PG 业务库到 MySQL 的离线预检查与单事务全量导入工具 |
| `cmd/analytics-backfill` | 将现有 MySQL 业务数据批量加入 PG 分析回填队列 |
| `migrations` / `VERSION` | 编译期通过 `embed` 内嵌的 SQL 迁移脚本与统一版本号文件 |

`internal/db` 中的 SQL 适配层仅覆盖本项目用到的查询语法。服务端启动时显式连接 MySQL 主库，当 MySQL 发生故障时不会将业务读写切换至 PostgreSQL。

## 运行与环境变量配置

MySQL 驱动默认启用客户端参数插值（可通过在 DSN 中显式指定 `interpolateParams=false` 切回二进制预编译语句）。在 `backend` 目录下配置环境变量即可直接启动服务：

```powershell
$env:MYSQL_DSN='suining:替换密码@tcp(127.0.0.1:13306)/suining'
$env:PG_ANALYTICS_URL='postgres://suining_analytics:替换密码@127.0.0.1:15432/suining_analytics?sslmode=disable'
$env:ADDR='127.0.0.1:8795'
$env:DATA_ENCRYPTION_KEY_FILE='替换为已备份的持久化密钥文件路径'
go run ./cmd/server
```

| 环境变量 | 说明 |
| --- | --- |
| `MYSQL_DSN` | 包含数据库名的 MySQL 连接串，必填，无默认密码 |
| `MYSQL_MAX_IDLE_CONNS` | MySQL 业务连接池最大空闲连接数，默认 20（取值范围 0 至 20；若数据库连接数受限可设为 5） |
| `HEALTH_LATE_DEVICE_LOCK` | 实验性健康写入加锁顺序开关，默认关闭；仅可在所有写入实例停机排空后统一设为 `1`，切勿在滚动升级中混用 |
| `PG_ANALYTICS_URL` | PostgreSQL 分析库连接串；留空时不影响核心业务读写 |
| `ADDR` | 监听地址，直接启动时默认为 `127.0.0.1:8080`，与前端 Vite 联调时建议设为 `127.0.0.1:8795` |
| `IMAGE_DIR` | 上传图片保存路径，直接启动时默认为 `./data/images` |
| `SESSION_TTL_HOURS` | 登录会话有效期，默认 12 小时 |
| `SESSION_BACKEND` | 会话存储后端，默认 `database`；`memory` 仅用于单实例本地开发 |
| `HEALTH_HOT_CACHE_MIB` | 健康统计进程内缓存上限，默认 16 MiB（范围 0 至 256，`0` 表示禁用） |
| `DATA_ENCRYPTION_KEY` / `DATA_ENCRYPTION_KEY_FILE` | 32 字节随机密钥的 Base64 字符串或密钥文件路径，多实例部署时必须共享同一密钥并持久保存 |
| `METRICS_TOKEN` / `METRICS_TOKEN_FILE` | 访问 `/metrics` 接口所需的 Bearer 令牌，长度至少 24 个字符 |
| `DESKTOP_TOKEN` | 桌面端用于首次初始化与安全退出的随机令牌，仅在原生进程与 Go 之间传递 |

在桌面模式下，Rust 主进程会将 `SUINING_MYSQL_DSN`、`SUINING_PG_ANALYTICS_URL` 及 `DATA_ENCRYPTION_KEY_FILE` 传递给 Go 子进程，同时注入动态回环端口、本地数据目录、随机令牌和父进程 PID，详见[桌面整合细则](../../docs/ARCHITECTURE.md)。本地开发也可直接运行仓库根目录的 `python3 scripts/start-platform.py` 自动拉起容器环境。

执行桌面完整打包前，需先在 `frontend` 目录运行 `npm run build`，将生成的前端静态资源复制到 `internal/handlers/web` 供 Go 编译内嵌。

## 健康统计缓存与数据一致性

`POST /api/v1/health/statistics` 使用进程内 LRU 缓存加速重复查询，按 `(elder_id, date_from, date_to)` 隔离键空间，最多缓存 2,048 项（单项上限 64 KiB，TTL 为 30 秒）。

每次统计请求都会先校验权限，并从 MySQL 读取对应长者的最新数据修订号（revision）。一旦任何实例通过 `health.SaveBatch` 写入、修改或删除了健康样本，修订号即会递增，使旧缓存自然失效并重新计算，无需跨实例网络广播。可通过 `/metrics` 中的 `suining_health_cache_hits_total` 与 `suining_health_cache_misses_total` 观察缓存命中情况。

## 健康写入加锁顺序（实验性）

默认情况下，`healthBatch` 遵循 `device 行锁 -> health_owner_clock 行锁 -> 样本与历史事务提交` 的加锁顺序，确保单设备高频并发上报与告警规则判定时的数据一致性。

当启用 `HEALTH_LATE_DEVICE_LOCK=1` 时，加锁顺序反转会变为 `health_owner_clock -> device`。为避免死锁，切勿让新旧两种加锁顺序的实例同时运行，切换前必须先停机排空所有应用实例。相关压测数据见 [MySQL 8.4 行锁隔离实验记录](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/397637821)。

## Go 运行时内存与容器 CPU 调度

- 规则判定生成的证据 JSON 按单次设备上报调用复用 512 B 私有缓冲区，仅在 `tx.Exec` 同步返回后回收复用。
- 在 Linux cgroup v2 容器环境中，设置 `SUINING_RUNTIME_AUTOTUNE=1` 可让进程在启动时自动读取 `cpu.max` 与 `memory.max` 并调整运行参数（若管理员已显式设置 `GOMAXPROCS` 或 `GOMEMLIMIT`，则以显式配置为准）。
- 可在仓库根目录运行 `python3 scripts/go123-memory-scheduling.py`，离线验证健康规则逻辑与 `GOMAXPROCS` 配额计算。

## 事务流转与存量数据迁移

建档、设备绑定、SOS 求助、工单处置与主管复核均在 MySQL 事务内完成。MySQL 触发器在同一事务中写入分析事件 outbox，确保业务回滚不会产生脏分析事件。后台 worker 异步消费 outbox 并幂等写入 PostgreSQL 的 `suining_analytics.record` 表及 `daily_health`、`care_summary` 视图。

服务端启动时会自动执行内嵌的 MySQL 迁移脚本（也可手动运行 `go run ./cmd/migrate`）。如需将旧版 PostgreSQL 业务库的数据迁移至 MySQL，请先停止服务写入并做好备份，配置 `LEGACY_POSTGRES_URL` 与 `MYSQL_DSN` 后执行：

```powershell
# 仅执行只读预检查
go run ./cmd/transfer-postgres
# 在单个目标事务中正式导入空 MySQL 库
go run ./cmd/transfer-postgres --apply
```

该工具支持旧版 PG 0001 至 0008 schema，导入时会保留原有记录 ID、哈希值与空值，并逐表核对行数。

## 本地测试与回归

```powershell
go test -count=1 ./...
go vet ./...
# 在独立测试库上运行完整竞态与集成测试（切勿指向生产库）
$env:TEST_MYSQL_DSN='测试MySQL原生DSN'
$env:TEST_DATABASE_URL='测试PG连接URL'
go test -race -count=1 -timeout 240s ./...
```

相关性能基线与故障排查文档：
- [双库混合压测与冷热缓存报告](../../docs/performance/2026-10-22-backend-hybrid/README.md)
- [微基准与耐久测试索引](../../docs/performance/2026-10-21-go123/README.md)
- [MySQL 503 / 500 连接池与 SQL 排查指南](../../docs/MYSQL_503_500_CONTINUATION_20261021.md)

## 接口路由与会话约定（破坏性变更）

服务端仅公开 `/api/v1/` 路由前缀，已全面移除无版本号的 `/api/` 路径且不设回退别名。浏览器登录会话统一使用 HttpOnly Cookie `SUINING_SESSION`，客户端需全局处理 `401`、`403` 与 `503` 状态码。
