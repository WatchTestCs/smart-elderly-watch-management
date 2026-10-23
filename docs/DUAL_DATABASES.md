# MySQL 主库与 PostgreSQL 分析库

2026-10-19，2.0.6 整合源码。采用 **MySQL 核心业务 + PG 分析**，保留现有后台权限与远程健康采样能力；此次仅推送 preview 源码，不发布 Release，不改动本机已有 MySQL 服务或生产数据。先前本地安装包不含此次健康采样整合，使用时需从当前源码重建。

## 数据分工

| 存储 | 内容 | 可用性边界 |
| --- | --- | --- |
| MySQL / InnoDB | 账号、老人/家属/监护人、设备、阈值、健康原始记录、轨迹、告警、照护任务与审计、公告评论、通知和分析 outbox | 唯一业务主库；绑定、SOS、处置和复核使用单库事务 |
| PostgreSQL | 心率血压、睡眠步数、轨迹、照护任务和事件的异步分析投影；健康日统计和照护状态汇总 | 最终一致，断线不阻断核心业务；不是 MySQL 热备或自动故障切换目标 |
| 本地文件 | 图片、日志和运行发现文件 | 图片单独备份 |

PG 不复制姓名、手机号、账号密码、设备密钥、照护备注和完整审计正文；投影仍含业务 ID、健康及位置数据，需独立访问控制。业务 API 始终读写 MySQL。`health_sample`、`health_change` 与 `health_clock` 也在 MySQL；新采样接口不改写旧日报或告警，当前 PG 分析投影范围仍为下文列出的旧记录、轨迹与照护数据，不自动复制新采样表。

```mermaid
flowchart LR
  U[Vue / Tauri / 手表] --> G[Go API]
  G -->|单库事务| M[(MySQL 核心业务)]
  M -->|同事务触发器| O[(分析 outbox)]
  O -->|异步 / 幂等 / 重试| P[(PG 分析投影与视图)]
```

## 配置与部署

MySQL DSN 未显式设置时，当前后端使用驱动的 `interpolateParams=true`，减少逐次 prepare/execute/close。驱动负责绑定值的转义与保守回退；强制 utf8mb4、禁用多语句，拒绝插值不安全的 collation。需要二进制 prepare 兼容模式时，在 DSN 明确设置 `interpolateParams=false`。这不改变事务提交与持久性策略。SQL 中含字面量问号等情况仍可能由驱动退回 prepare。

开发者向 JSON 列传值应使用 `json.RawMessage` 或 JSON 文本字符串；普通 `[]byte` 保持二进制语义，用于 BLOB。JSON 与二进制不能混用。数据库参数中不自行拼接或转义用户值。

| 变量 | 使用位置 |
| --- | --- |
| `MYSQL_DSN` | Go 与迁移工具，go-sql-driver/mysql 原生 DSN，必须指定业务库 |
| `PG_ANALYTICS_URL` | Go，专用分析库的 PG URL |
| `SUINING_MYSQL_DSN` / `SUINING_PG_ANALYTICS_URL` | Tauri，将对应配置传给 Go |
| `LEGACY_POSTGRES_URL` | 离线导入工具的旧 PG 主库 |
| `TEST_MYSQL_DSN` / `TEST_DATABASE_URL` | 专用测试实例；测试创建并清理随机数据库或 schema |

在 backend 目录执行，私下替换占位密码：

```powershell
$env:MYSQL_DSN='suining:替换密码@tcp(127.0.0.1:13306)/suining'
$env:PG_ANALYTICS_URL='postgres://suining_analytics:替换密码@127.0.0.1:15432/suining_analytics?sslmode=disable'
go run ./cmd/server

$env:SUINING_MYSQL_DSN=$env:MYSQL_DSN
$env:SUINING_PG_ANALYTICS_URL=$env:PG_ANALYTICS_URL
```

无 TLS 示例仅用于受控本机连接；远程 MySQL 使用验证证书的 `tls=true`，远程 PG 配置 `sslmode=verify-full` 与信任证书。实际凭据不提交到源码、日志或截图。[MySQL 驱动配置](https://github.com/go-sql-driver/mysql)

服务强制参数绑定、utf8mb4、UTC 时间、时间值解析与严格 SQL 模式。保留微秒精度，避免刚入队的通知因时间截断而无法立即领取。SQL 适配覆盖本应用的重复参数、upsert、自增 ID、日期格式、拼接和锁语句，不是任意 PG SQL 的通用翻译器。

当前业务服务不读取旧的 `DATABASE_URL` / `SUINING_DATABASE_URL` 作为主库回退。旧 PG 迁移须显式运行 `go run ./cmd/migrate --legacy-postgres`，仅该工具读取 `DATABASE_URL`。

仓库提供 [Compose 模板](../deploy/dual-db.compose.yml) 和 [环境变量模板](../deploy/dual-db.env.example)。从仓库根目录执行：

```powershell
Copy-Item deploy/dual-db.env.example deploy/dual-db.env
# 私下填写三个不同的密码；dual-db.env 已加入 .gitignore。
docker compose --env-file deploy/dual-db.env -f deploy/dual-db.compose.yml up -d
```

模板提供 MySQL 8.4、PG 16、独立持久卷和健康检查，仅映射回环端口 13306、15432。应用另行启动。停止服务不要删除持久卷。

MySQL 账号首次迁移需要业务库 DDL、索引、外键与触发器创建权限；运行需要业务表和 outbox 读写权限。PG 分析账号需要在专用库创建 `suining_analytics` schema、表、索引、视图及投影读写权限，报表账号另行授权 SELECT。

## 初始化与旧库导入

MySQL 迁移使用连接级 `GET_LOCK` 和脚本 SHA-256；非空且无本项目迁移记录的库拒绝初始化。MySQL DDL 会隐式提交，因此采用可重复执行的语句、最后记录版本，不宣称整份 DDL 可回滚。[MySQL 官方说明](https://dev.mysql.com/doc/refman/8.4/en/implicit-commit.html)

PG 分析迁移使用事务及 advisory lock，只维护分析 schema，不修改旧业务 schema。

迁移旧 PG 前，备份旧库和图片，停止旧、新应用写入，准备本项目已迁移的空 MySQL：

```powershell
$env:LEGACY_POSTGRES_URL='旧版PG连接串'
$env:MYSQL_DSN='目标空MySQL的原生DSN'
go run ./cmd/transfer-postgres
go run ./cmd/transfer-postgres --apply
```

默认只预检查及输出逐表行数。`--apply` 使用旧 PG 的只读、可重复读快照，按外键顺序导入 MySQL 单个事务，保留 ID、密码哈希、设备关联、时间和 NULL；逐表核对行数后提交。失败时目标业务数据及分析事件一同回滚，源库只读，目标非空拒绝导入。目标建表是独立阶段，导入失败不会删除已创建的空表。

导入会生成分析事件，PG 恢复后补齐。图片需另行复制。本工具支持 PG 0001–0006 结构，以及带健康采样的 0007／0008；后两者同时导入采样、变更记录、修订时钟与设备绑定版本，不自动修复历史错误字段，也不直接支持旧 Java/H2。上线前核对行数、账号、设备归属、照护履历及图片，再切换配置。

## 同步、故障与重建

1. MySQL 触发器将新增、更新、删除与分析事件一起提交；业务回滚时事件回滚。
2. 后台向 PG 写入以 `source_id + entity + entity_id` 唯一的投影。PG 成功后才删除主库队列记录。
3. 在 PG 提交与主库确认之间崩溃会重放；`last_event_id` 忽略重复和旧事件。删除保留墓碑，旧事件不能使数据复活。
4. PG 失败保留队列并重连重试；主管诊断显示配置状态、可用性、积压与最近同步时间。MySQL 失败才阻断业务；失败业务写入不自动重放。

队列依赖 MySQL 磁盘，长期 PG 故障需监控积压和容量，不承诺无限离线。多实例共用主库稳定的 `analytics_source.source_id`。克隆、换主库或从旧快照恢复时，使用独立分析库重建，避免陈旧的大事件号压制恢复后的记录。

PG 丢失数据时，优先准备新的分析库、修改连接并重启，然后从当前主库回填：

```powershell
go run ./cmd/analytics-backfill --apply
```

回填按批锁定当前行、生成新事件，不修改业务记录；并发更新产生更晚的事件。不能把陈旧分析投影当作业务恢复依据。

分析位于 `suining_analytics.record`，提供两个视图：

```sql
SELECT * FROM suining_analytics.daily_health ORDER BY day DESC;
SELECT * FROM suining_analytics.care_summary ORDER BY source_id,state;
```

日统计分别统计心率/血压有效样本，NULL 不计入均值；照护汇总按状态分组。业务界面继续读主库当前数据。

## 本地验证和限制

独立 MySQL 8.0.46 与 PostgreSQL 16.4 实测：

- MySQL 上 Go 全套测试、竞争检测，覆盖原有照护、绑定、权限、通知与新增格式回归。
- 双库测试覆盖回滚不泄露事件、PG 失败保留、恢复补偿、重放、删除及旧事件防复活、投影回填。
- 迁移测试覆盖预检查不写入、ID/中文/NULL 保留、拒绝非空库和失败整笔回滚。
- 生产前端 + 真实 Go + MySQL：PG 全程停机时完成 15 项建档到独立复核检查。随后启动 PG，实查自动重连、队列归零、已复核任务与缺测指标一致。
- Vue 构建与浏览器回归检查双库提示、恢复及既有页面；Compose 做静态解析。未拉取模板镜像、未重建原生安装包。

MySQL 8.4 镜像、生产数据迁移、远程 TLS、真实硬件/短信以及长期大数据量未实测。原生环境变量桥已修改，未进行 Tauri 运行验收。

入口：[数据库适配](../智慧养老手表管理系统/backend/internal/db/database.go)、[同步](../智慧养老手表管理系统/backend/internal/analytics/sync.go)、[导入工具](../智慧养老手表管理系统/backend/cmd/transfer-postgres/main.go)、[回填工具](../智慧养老手表管理系统/backend/cmd/analytics-backfill/main.go)。

### 复跑与 CI

在 backend 同时配置专用 `TEST_MYSQL_DSN` 和 `TEST_DATABASE_URL` 后执行 `go test -race -count=1 -timeout 240s ./...`、`go vet ./...`；MySQL 测试账号需要创建/删除随机测试数据库的权限。缺少变量时对应集成测试会跳过，不应把跳过视为双库验收通过。

前端在 oldIot 执行 `npm run build`、`npm --prefix qa test`。真实后端浏览器脚本需独立空 MySQL、启动后的服务 URL、桌面测试令牌与 `CARE_E2E_TEST_ONLY=1`；仅支持 `qa_arch_` 前缀测试库。源码中的后端 CI 与迁移工作流已增加 MySQL 服务和双库测试变量；本轮仅修改及静态校验，没有运行远程 Actions。
