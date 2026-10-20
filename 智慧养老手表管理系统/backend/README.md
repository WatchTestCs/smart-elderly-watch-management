# 岁宁 Go 后端 · 2.0.6

Go 服务负责业务 API、设备接入、会话、图片、内嵌前端、照护与通知作业，以及分析同步。**MySQL 是唯一业务主库，PostgreSQL 是异步分析库**；Windows 由 Tauri 管理 Go 生命周期，也可独立部署服务用于受控设备接入和长期值守。

[整体架构](../../README.md#系统架构) · [双库部署与迁移](../../docs/DUAL_DATABASES.md) · [照护部署](../../docs/CARE_SERVICE_DEPLOYMENT.md)

## 技术与模块

Go 模块名为 `careold-go`，声明最低 Go 1.24.0，本地 2.0.6 使用 Go 1.27.2 构建。HTTP 使用标准库 `net/http`，主库存取使用 `database/sql` 与 `go-sql-driver/mysql`，PG 投影及旧库工具使用 `pgx/v5`。当前没有独立消息中间件或分布式事务协调器。

| 位置 | 职责 |
| --- | --- |
| `cmd/server` | HTTP、后台连接恢复、迁移发布、桌面就绪与父进程监测 |
| `internal/handlers` | 账号、人员、设备、健康、公告、照护与诊断 API；图片与静态资源 |
| `internal/auth` / `session` / `csrf` / `ratelimit` | 角色、BCrypt、内存会话、CSRF 与限流 |
| `internal/care` | 通知投递、超时升级与照护后台作业 |
| `internal/db` | MySQL 连接与 SQL 适配、事务、迁移、旧 PG 兼容 |
| `internal/analytics` | PG schema、outbox 同步、事件去重、墓碑与回填 |
| `cmd/migrate` | 默认执行 MySQL 迁移；显式参数支持旧 PG 迁移 |
| `cmd/transfer-postgres` | 旧 PG 到空 MySQL 的离线预检查与事务导入 |
| `cmd/analytics-backfill` | 将当前 MySQL 数据加入分析回填队列 |
| `migrations` / `VERSION` | 编译内嵌 SQL 与统一产品版本 |

SQL 适配覆盖本应用使用的语法，不是任意 PostgreSQL SQL 的通用翻译层。应用服务器显式打开 MySQL，主库故障不会自动切到 PG。

## 运行配置

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
| `PG_ANALYTICS_URL` | 独立 PG 分析库 URL；不配置仍可运行核心业务 |
| `ADDR` | 直接启动默认 `127.0.0.1:8080`；前端开发代理使用 8795 |
| `IMAGE_DIR` | 直接启动默认 `./data/images` |
| `SESSION_TTL_HOURS` | 会话期限，默认 12 小时 |
| `SESSION_BACKEND` | 默认 database；memory 仅限显式选择的单实例开发 |
| `DATA_ENCRYPTION_KEY` / `DATA_ENCRYPTION_KEY_FILE` | 必须提供 base64 编码的 32 字节随机密钥；所有实例共享，不得每次启动更换 |
| `METRICS_TOKEN` / `METRICS_TOKEN_FILE` | `/metrics` Bearer 凭据，至少 24 字符 |
| `DESKTOP_TOKEN` | 初始化与桌面退出的随机令牌，不交给网页 |

桌面由 Rust 将 `SUINING_MYSQL_DSN` / `SUINING_PG_ANALYTICS_URL` 转交给 Go，并传入回环动态端口、本地目录、随机令牌和父 PID。旧 `DATABASE_URL` / `SUINING_DATABASE_URL` 不作为业务主库配置。示例只用于受控本机；远程连接使用验证证书的 TLS，不提交真实凭据。

桌面启动前也需在父进程环境中设置 `DATA_ENCRYPTION_KEY_FILE`，由后端继承；未配置时数据库 API 不开放。仓库根目录的 `python scripts/start-platform.py --validate` 可一次生成开发部署密钥；不要提交生成的 secrets。升级、历史备注回填、DDL 回滚与具体加密范围见 [平台改造说明](../../docs/PLATFORM_UPGRADE_2.0.6.md)。

前端 `npm run build` 的结果复制到 `internal/handlers/web` 后由 Go 编译内嵌。源码归档只保留占位资源；纯 API 开发可以直接运行，桌面发布需执行 [完整构建](../README.md#windows-完整构建)。

## 事务与分析数据流

业务处理器在 MySQL 中完成建档、绑定、SOS、处置和复核。MySQL 触发器将有关健康、活动、轨迹及照护的分析事件与业务变更一同提交；回滚不会泄露分析事件。Go 分析作业写入 PG 后才确认队列，重复投递按事件号去重，删除墓碑防止旧事件复活。

PG 保存 `suining_analytics.record` 投影，并提供 `daily_health`、`care_summary` 视图；业务页面仍读取 MySQL 当前数据。PG 未配置或断线时队列继续保留，恢复后自动补齐，需监控主库容量。分析投影不能作为主库热备或自动切换目标。

照护任务支持接单、开始处置、记录措施、另一名主管独立复核，状态与履历由事务维护；复核关闭更新对应告警。护理员／审计员岗位限制覆盖照护功能，其他后台权限保持现状。通知发送、送达回执与人员到场分别确认。

## 迁移与存量数据

服务启动自动迁移 MySQL，也可运行 `go run ./cmd/migrate`。SQL 编入程序，不依赖当前工作目录。MySQL 使用连接级锁和脚本校验和；DDL 会隐式提交，采用可重复语句和最终版本记录，不承诺整份 DDL 事务回滚。PG 分析迁移使用事务和 advisory lock，仅维护分析 schema。

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

## 故障与安全边界

- HTTP、静态页面和桌面就绪不等待数据库；MySQL 迁移成功后才发布业务 API，失败后后台重试。
- `GET /healthz` 只代表进程存活；`GET /restful/service/status` 返回主库就绪及分析配置、连接状态，不暴露连接串。MySQL 未就绪时业务 API 返回 `503 DATABASE_UNAVAILABLE`；探测前正在执行的 SQL 也可能先失败。
- MySQL 恢复后检查会话；失败业务写入不自动重放。分析队列只重试已经提交的分析事件。
- 首位管理员必须经桌面令牌初始化，无默认账号；普通写接口校验会话、角色和 CSRF，设备接口单独鉴权。会话在内存中，重启失效。
- 桌面本地服务随窗口退出。长期值守需常驻部署、通知网关和人工流程；图片需与 MySQL 分别备份。
