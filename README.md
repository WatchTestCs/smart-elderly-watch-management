<p align="center"><img src="智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-192.webp" width="88" height="88" alt="岁宁标志"></p>

# 岁宁 · 智慧养老管理

**简体中文** | [English](README.en.md)

[![License: MIT](https://img.shields.io/badge/license-MIT-235442)](LICENSE) [![Preview: 2.1.0](https://img.shields.io/badge/preview-2.1.0-9b7a40)](docs/releases/2.1.0.md) [![Go: 1.23+](https://img.shields.io/badge/Go-1.23%2B-00ADD8)](智慧养老手表管理系统/backend/go.mod)

**岁月有伴，日常安宁。**

岁宁面向照护机构与家属，提供长者档案、手表设备、健康与活动记录、位置告警、照护任务和公告互动。当前 2.1.0 采用 **Tauri 2 / Rust 桌面 + Vue 3 / Vite 界面 + Go 业务服务 + MySQL 主库 + PostgreSQL 分析库**，数据库由部署方独立管理。

[快速开始](#开始使用) · [功能与照护流程](#主要功能) · [系统架构](#系统架构) · [源码与构建](#源码与构建) · [测试与性能证据](#近期后端验证与性能证据) · [下载与历史附件](docs/DOWNLOADS.md) · [双库部署与迁移](docs/DUAL_DATABASES.md)

## 当前源码与适用范围

> **2.1.0 完整交付：**[v2.1.0 Release](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.1.0) 提供 Windows x64 安装版、免安装版及完整源码。精确提交、构建与打包检查见随包 `release-manifest.json`；未签名，真实设备和生产环境仍须独立验收。

| 关心的问题 | 当前结论 | 从哪里核对 |
| --- | --- | --- |
| 获取最新开发源码 | 使用 `preview` 分支；分支 ZIP 会随提交变化 | [浏览源码](https://github.com/WatchTestCs/smart-elderly-watch-management/tree/preview) · [下载源码 ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/archive/refs/heads/preview.zip) |
| 对外版本与安装包 | **2.1.0 安装版、免安装版和源码**，三者对应同一提交 | [v2.1.0 Release](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.1.0) · [下载说明](docs/DOWNLOADS.md) |
| Go 版本 | `backend/go.mod` 声明 **Go 1.23+**；CI 覆盖 Go 1.23.2 与 1.27.2 | [go.mod](智慧养老手表管理系统/backend/go.mod) · [最新已核实 CI 运行](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39634129082) |
| 核心数据库 | MySQL 是业务唯一主库，PostgreSQL 是可重建的异步分析投影 | [双库指南](docs/DUAL_DATABASES.md) |
| 真实长时间负载 | 历史两轮 CI 耐久测试失败；后续本地修复版完成 20 分钟双库混合压测，9,484 批写入成功，HTTP 500/503 均为 0；当前合并源码未重跑 20 分钟 | [最新结果与源码边界](docs/performance/2026-10-22-backend-hybrid/README.md) · [历史记录](#近期后端验证与性能证据) |
| 生产与安全边界 | 非医疗器械认证、未经过真实设备和完整通知网关端到端验收；不能以系统状态替代人工应急响应 | [照护部署](docs/CARE_SERVICE_DEPLOYMENT.md) · [测试说明](docs/TESTING.md) |

**阅读顺序：**初次使用看 [开始使用](#开始使用)，维护后端看 [后端说明](智慧养老手表管理系统/backend/README.md)，部署值守看 [照护部署](docs/CARE_SERVICE_DEPLOYMENT.md)，审查性能先看 [最新修复与冷热缓存报告](docs/performance/2026-10-22-backend-hybrid/README.md)，再查 [历史证据](docs/performance/2026-10-21-go123/README.md)。

### preview 后续修复：配置诊断与 SQL 翻译

当前 `preview` 在 2.1.0 发布提交之后增加配置诊断：`SESSION_BACKEND` 无效，或默认 `database` 会话模式缺少有效 `DATA_ENCRYPTION_KEY` 时，界面仍可打开，但状态接口返回 `configError`，页面明确提示修正配置并重启；不再把这种错误描述成正在自动重连。数据库会话需要 base64 编码的 32 字节密钥。只有明确使用 `SESSION_BACKEND=memory` 才适用于无密钥的单实例开发。

SQL 翻译同时修复无空格 `~` 正则匹配，并保留引号内的 `now()`、`~` 和占位符文本。这些是发布后的源码更改，**现有 v2.1.0 附件未替换**。

## 仓库目录与根文件

| 路径 | 用途 |
| --- | --- |
| [.github/](.github/) | 漏洞披露、依赖更新、PR 模板和自动检查工作流。 |
| [deploy/](deploy/) | 双库部署模板、[完整平台 Compose](deploy/platform.compose.yml)（Go 应用、Nginx、Prometheus）及运维配置；[Dockerfile](deploy/Dockerfile) 也位于此目录。 |
| [docs/](docs/) | 使用、[架构总览](docs/architecture/overview.md)、迁移、照护运维、历史版本与[逐文件职责索引](docs/FILES.md)。 |
| [tests/ui/](tests/ui/) | 无外部依赖的浏览器可访问性冒烟检查。 |
| [智慧养老手表管理系统/](智慧养老手表管理系统/README.md) | 可构建的桌面工程：Tauri、Go 后端、Vue 前端和手表模拟器。 |
| [.gitignore](.gitignore) | 排除依赖缓存、数据库、构建产物与私钥。 |
| [CHANGELOG.md](CHANGELOG.md) | 按版本记录功能变化、修复与兼容性。 |
| [CONTRIBUTING.md](CONTRIBUTING.md) | 贡献流程、分支和验证约定。 |
| [LICENSE](LICENSE) | 仓库 MIT 许可和版权声明。 |
| [README.en.md](README.en.md) | 英文项目介绍、技术栈、下载和部署入口。 |
| [README.md](README.md) | 中文项目首页与当前源码入口。 |

2026-10-20 的历史性能报告和原始数据统一归档在 [Go 1.23 性能资料](docs/performance/2026-10-20-go123/README.md)，根目录仅保留项目入口与仓库级文件。

## 2.1.0 安装版、免安装版与源码

[v2.1.0 Release](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.1.0) 提供 `Suining-Setup-2.1.0-win-x64.exe`、`Suining-Portable-2.1.0-win-x64.zip`、`Suining-Source-2.1.0.zip`、SHA-256 和来源清单。继承双库、照护和档案整合，纳入后续 Go 修复及 PR #24／#25 的分析排空、日缓存和 MySQL 空闲池优化。[升级与验证边界](docs/releases/2.1.0.md)。

安装版复用系统 WebView2，仅缺失时联网补装；免安装版需已有运行时，解压后运行 `suining.exe` 并保留同目录 `suining-server.exe`。数据库独立部署，安装不会自动导入旧数据。当前包未签名。

`MYSQL_MAX_IDLE_CONNS` 默认 20，允许 0–20；设为 5 并重启可恢复旧空闲池行为。主应用连接数仍最多 20，健康探测另占 1 个；多实例按数据库资源预算配置。性能收益只对应报告中的查询负载。

最新开发源码仍查看 [preview](https://github.com/WatchTestCs/smart-elderly-watch-management/tree/preview)，固定发布版使用 Release 源码包。历史 2.0.6 只有源码，2.0.2 使用旧 PG 单库，2.0.0 使用早期桌面栈；历史 Linux 附件没有重编译为 2.1.0。完整索引见 [下载与历史附件](docs/DOWNLOADS.md)。

## 主要功能

| 场景 | 当前功能 |
| --- | --- |
| 长者与家庭 | 长者、监护人、家属档案，邀请码注册与关联家人 |
| 手表设备 | 登记、绑定、解绑、设备鉴权、在线状态、监测阈值 |
| 健康与日常 | 心率、单项血压、睡眠、步数和生活信息查询 |
| 位置与告警 | 最近上报位置、异常记录、SOS 与任务履历 |
| 照护协作 | 接单、开始处置、登记措施、独立复核及值守诊断 |
| 公告与互动 | 图文公告、评论与后台管理 |
| 桌面体验 | 大字、键盘操作、全屏、窄屏卡片与宽屏布局 |

主管、护理员、审计员的岗位限制覆盖照护任务，保留现有其他后台权限；家属仅访问关联家人的记录。健康、位置与在线状态依赖有效上报，缺少新记录不代表状态正常。地图采用 Leaflet 与 OpenStreetMap 瓦片，显示依赖相应网络服务。

## 从建档到照护完成

1. **机构准备。** 部署 MySQL、配置通知与常驻服务，首次创建管理员。主管可分配照护岗位；没有岗位记录的旧管理员保留主管待遇，避免升级后被锁出照护功能。
2. **建立人员关系。** 先创建长者与监护人资料，再登记、绑定手表。家属通过邀请码注册和关联；普通档案编辑不会代替设备改绑。
3. **接收有效记录。** 设备通过独立密钥上报。新健康采样接口按设备、长者和绑定版本区分记录，支持修订与重复上报；旧上报和旧图表仍沿用原路径。
4. **处理求助。** SOS 进入任务列表，由有权限的人员接单、开始处置并登记措施。通知已发送或已送达不等于任务已处置。
5. **独立复核。** 另一名主管核对处置后关闭任务，同步清除关联告警；后来新增的其他告警继续保留，任务和事件履历可分页查询。
6. **值守与复盘。** 查看主库就绪、通知积压、超时任务及分析同步状态。PG 用于异步分析，当前业务界面和新健康统计接口仍读 MySQL。

| 身份或照护岗位 | 可执行范围 |
| --- | --- |
| 主管 | 照护处置、岗位配置、服务诊断与复核；处置人不能复核自己的任务 |
| 护理员 | 照护接单、开始和登记处置，不能独立执行主管复核 |
| 审计员 | 读取照护任务与履历，不能执行任务状态变更 |
| 家属 | 访问关联家人的资料和记录，不接单或关闭机构照护任务 |

岗位限制目前只覆盖照护模块，其他后台功能保留原权限范围。配置岗位并不等于已经对全部档案、设备、公告实现细分授权。

## 技术栈

下表为当前源码依赖；精确版本以各模块锁文件为准。

| 层次 | 技术 | 职责 |
| --- | --- | --- |
| Windows 桌面 | Tauri 2、Rust、WebView2 | 单实例、窗口、受限原生命令、Go 子进程校验与生命周期 |
| 前端 | Vue 3.5、Vue Router 4、Element Plus 2、Vite 8 | 管理端与家属端路由、表单、响应式界面和生产资源 |
| 请求与可视化 | Axios 1、ECharts 6、Leaflet 1.9 | API 请求、健康图表和位置展示 |
| 业务服务 | Go、标准库 `net/http`、`database/sql` | 业务 API、设备接入、会话、CSRF、图片与后台作业 |
| 主库存取 | MySQL / InnoDB、`go-sql-driver/mysql` | 核心业务事务、关系约束和持久化事件队列 |
| 分析存取 | PostgreSQL、`pgx/v5` | 可重建的分析投影、健康日统计与照护状态汇总视图 |
| 构建与交付 | NSIS、npm / Cargo 锁文件、Go modules、SHA-256 | Windows 安装包、依赖固定、版本与完整性校验 |
| 验证工具 | Go 测试 / 竞争检测、Playwright、手表模拟器 | 业务回归、浏览器验收和协议模拟 |

这是同一 Go 服务内组织处理器与后台作业的架构。现有源码不依赖 Java、Spring Boot、Electron、H2、Vue 2 或旧 Webpack 构建链；`oldIot` 与 Go 模块名 `careold-go` 保留为历史命名。

## 系统架构

```mermaid
flowchart LR
  subgraph Client["Windows 桌面"]
    R["Tauri / Rust"]
    V["Vue 3 / WebView2"]
    R -->|"创建窗口"| V
    V -->|"受限初始化 IPC"| R
  end
  subgraph Server["Go 服务进程"]
    G["HTTP API / 内嵌前端"]
    C["照护与通知作业"]
    A["分析同步作业"]
  end
  R -->|"SHA-256 校验、启动与退出"| G
  V -->|"同源 HTTP / 会话 / CSRF"| G
  G -->|"核心业务事务"| M[("MySQL / InnoDB")]
  C --> M
  M -->|"业务事务内触发器写入"| O[("MySQL 分析 outbox")]
  O -->|"读取已提交事件"| A
  A -->|"异步投影、幂等重试"| P[("PostgreSQL 分析库")]
  G --> F["本地图片与日志"]
```

Windows 默认将 Go 作为本地子进程启动，监听回环动态端口。Go 先提供页面与状态，再后台连接 MySQL 并执行内嵌迁移；Rust 核验就绪文件的 PID 和端口后加载本机页面。关闭窗口时管理 Go 退出，Go 同时监测父进程。

Vue 通过 HTTP 使用 Go API，不直接连接数据库。首次管理员初始化通过受限 Tauri IPC，由 Rust 附加桌面令牌；网页不持有该令牌或数据库连接串。普通业务写入使用会话、角色和 CSRF 校验，设备接口使用独立设备鉴权。会话后端默认配置为 `SESSION_BACKEND=database`；仅在明确指定的单实例开发场景才使用 `memory`。重启、跨实例登录状态与故障恢复需按部署模式分别验收。

### 双库数据流与故障行为

| 存储 | 保存内容 | 恢复与一致性 |
| --- | --- | --- |
| MySQL 主库 | 账号、人员与设备、健康原始记录、轨迹、告警、照护、审计、公告评论、通知及分析 outbox | 唯一业务数据源；业务 API 始终读写 MySQL |
| Go 进程内热缓存 | 重复健康统计的序列化结果；默认 16 MiB、最多 2,048 项、30 秒 TTL | 每次鉴权并核验数据库修订号；淘汰或重启后从 MySQL 重建，全部原始历史继续持久化 |
| PostgreSQL 分析库 | 健康、活动、轨迹、照护任务与事件的投影；`daily_health`、`care_summary` 视图 | 最终一致，可由 MySQL 回填重建，不能替代主库备份 |
| 本地文件 | 图片、日志和运行发现文件 | 图片与数据库分别备份；运行文件不是业务数据源 |

MySQL 触发器在业务事务内写入分析事件；Go 同步作业写入 PG 成功后才确认主库队列，允许至少一次投递。投影用事件号去重并忽略旧事件，删除保留墓碑。PG 不接收姓名、手机号、密码、设备密钥、照护备注或完整审计正文，但投影中的健康与位置数据仍需访问控制。

| 情况 | 实际行为 |
| --- | --- |
| MySQL 未连接或迁移未完成 | 界面仍打开，业务 API 返回 `503 DATABASE_UNAVAILABLE` |
| MySQL 连接恢复 | 后台重新检查数据库与会话，失效会话需重新登录 |
| PG 未配置或不可用 | 核心业务继续，已提交分析事件保留在 MySQL，需监控队列与磁盘容量 |
| PG 恢复 | 自动重连并补齐分析投影，不切换业务主库 |
| 用户保存等写入失败 | 保留输入、提示重试，不自动重放失败业务请求 |

`GET /healthz` 只代表 HTTP 进程存活；`GET /restful/service/status` 返回主库就绪及分析配置、连接状态。主管诊断另有积压量与最近同步时间。通知回执、人员到场和照护完成仍需分别确认。

完整进程协议见 [实现架构](docs/ARCHITECTURE.md)，双库权限、队列恢复和迁移细节见 [DUAL_DATABASES](docs/DUAL_DATABASES.md)。

## 开始使用

1. 从 [v2.1.0 Release](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.1.0) 下载安装版或免安装版；完整解压免安装包后运行 `suining.exe`，保留同目录 `suining-server.exe`。最终用户无需 Go、Rust 或 Node.js。Setup 缺少 WebView2 时联网补装；免安装版需要已有 WebView2。
2. 部署独立 MySQL 业务库，按需启用 PG 分析库。安装包不包含数据库服务。可参考 [双库 Compose 模板](deploy/dual-db.compose.yml) 与 [环境模板](deploy/dual-db.env.example)；模板仅映射回环端口，应用需另行启动。
3. 配置桌面环境变量后启动软件。以下仅为受控本机示例，密码需私下替换；跨机器连接需验证 TLS。

```powershell
$env:SUINING_MYSQL_DSN='suining:替换密码@tcp(127.0.0.1:13306)/suining'
$env:SUINING_PG_ANALYTICS_URL='postgres://suining_analytics:替换密码@127.0.0.1:15432/suining_analytics?sslmode=disable'
& '.\suining.exe'
```

4. MySQL 迁移完成后，在空库初始化首位管理员，**没有默认手机号或通用密码**。家属通过邀请码注册。
5. 先创建长者档案，再登记、绑定手表。默认桌面回环端口无法供互联网手表直接访问；实际接入需受控网关或独立 Go 服务部署。长期值守需要常驻后台服务、通知网关和人工响应流程，见 [照护部署](docs/CARE_SERVICE_DEPLOYMENT.md)。

直接运行 Go 时使用 `MYSQL_DSN`、`PG_ANALYTICS_URL`，默认 `ADDR=127.0.0.1:8080`；桌面使用上述 `SUINING_` 变量并传给 Go。配置修改后重启。当前业务服务不读取旧 `DATABASE_URL` / `SUINING_DATABASE_URL` 作为主库回退。

### 一键启动隔离的本地平台（用于开发和验证）

仓库除历史 [双库模板](deploy/dual-db.compose.yml) 外，还包含 [platform.compose.yml](deploy/platform.compose.yml)：MySQL 8.4、PostgreSQL 16、Go 应用、Nginx 网关与 Prometheus。需要安装 Docker Compose、Python 3，并确保相应镜像可获取。以下命令在**开发机器**的仓库根目录执行：

```sh
# 生成尚不存在的本地测试密钥，并校验完整平台 Compose
python3 scripts/start-platform.py --validate

# 在已有 Docker Compose 环境下构建并启动平台
python3 scripts/start-platform.py
```

启动脚本会在 `deploy/secrets/`、`deploy/platform.env` 创建本地敏感配置（已有文件不会自动覆盖），**即使带 `--validate` 也会创建缺失的文件**，运行前应了解这一副作用。默认网关和指标页面仅映射到主机回环地址；外部设备接入仍需单独配置可信反向代理、TLS 与网络访问控制。此命令不能替代备份、实际通知网关、照护值班制度或生产部署验收。

开发配置和最小权限要求见 [平台迁移说明](docs/PLATFORM_UPGRADE_2.0.6.md) 和 [后端 README](智慧养老手表管理系统/backend/README.md)。不应把自动生成的密钥、明文环境文件或真实健康记录提交到 Git。

### 选择部署方式

| 方式 | 适用情况 | 必须另外准备 |
| --- | --- | --- |
| Windows 桌面＋独立数据库 | 本机管理和查看；Rust 管理本地 Go 服务 | WebView2、MySQL；需要分析时配置 PG |
| 独立 Go 常驻服务＋受控接入 | 设备持续上报、通知投递和长期值守 | 服务监护、网络边界、TLS、通知网关、数据库和图片备份 |
| Vite＋Go 分开开发 | 页面与接口联调 | Node.js、Go、专用开发库；首次初始化仍需原生链路 |

桌面默认不作为远程 Go 服务的通用客户端；不能只修改数据库地址就认为手表网关或常驻值守已配置完成。关闭桌面会结束它管理的本地 Go 子进程。

### 从旧 PG 业务库升级

先备份数据库和图片，停止新旧应用写入，准备已迁移的空 MySQL。在 `backend` 中配置 `LEGACY_POSTGRES_URL` 和 `MYSQL_DSN`，先执行 `go run ./cmd/transfer-postgres` 预检查，再执行 `go run ./cmd/transfer-postgres --apply`。工具只读源库，拒绝非空目标，导入失败时目标业务数据及分析事件一同回滚；建表是独立阶段，图片另行复制。它支持完整 PG 0001–0006 结构，也保留 0007／0008 的健康采样、修订游标与绑定版本，不直接迁移 Java/H2，也不自动纠正历史错误数据。

PG 分析库丢失时，可在准备新分析库并修改配置后运行 `go run ./cmd/analytics-backfill --apply` 生成回填事件。安装软件不会自动搬迁旧业务库，具体核对与切换步骤见 [迁移说明](docs/DUAL_DATABASES.md)。

## 源码与构建

```text
智慧养老手表管理系统/
├── desktop/      Tauri / Rust、能力配置、版本校验与 Windows 构建
├── backend/      Go API、照护与分析作业、内嵌 SQL、迁移和回填工具
├── oldIot/       Vue 3 / Vite、页面组件、本地字体与浏览器 QA
└── watch-sim/    手表接口与端口发现模拟工具
deploy/          MySQL / PG Compose 与环境变量模板
docs/            配置、架构、验收、版本、备份与运维文档
```

Windows 构建需要 PowerShell 7、Node.js（`^20.19.0 || >=22.12.0`）、Go、Rust 和 MSVC / Windows SDK。**当前 Go 模块最低版本为 1.23.0**，已有 CI 验证 Go 1.23.2 与 1.27.2；先前 2.0.6 本地构建使用 Go 1.27.2。Rust 包声明最低 1.85。从工程目录执行：

```powershell
cd 智慧养老手表管理系统
./desktop/validate-version.ps1
./desktop/build.ps1
```

构建顺序为 **Vue 生产资源 → Go 内嵌资源、测试与编译 → sidecar SHA-256 → Tauri 原生程序 → NSIS 安装包**。脚本支持 `CARGO_TARGET_DIR`，产物在工程目录 `outputs/v2.1.0-preview`。默认构建不签名；可选 `-SelfSign` 仅用于本机自签，不能提供公开受信任签名。运行 `desktop/package-release.ps1` 生成免安装 ZIP、源码 ZIP、来源清单并检查安装和启动。开发代理、真实数据库测试要求见 [工程 README](智慧养老手表管理系统/README.md) 和 [后端 README](智慧养老手表管理系统/backend/README.md)。

## 近期后端验证与性能证据

本节记录 **2026-10-21–22 取得的可核实证据**，与历史 2.0.6 Windows 安装交付分开。这里的指标来自不同源码、环境和负载，**不可相加或直接用于生产 SLA**；当前合并源码的定向验证不等于重新完成长压测。

| 证据 | 运行环境与结果 | 可证明与不能证明的事 |
| --- | --- | --- |
| Go 双版本后端 CI | [Preview CI 历史通过运行](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39623700602)，Go **1.23.2 / 1.27.2**，包含 MySQL 8.4 与 PG 16 测试服务 | 可以证明该提交在所配置 CI 用例下通过；不能证明所有生产场景无错 |
| 规则、健康与 SQL 适配微基准 | Go 1.23.2，4 CPU 配额的隔离计算测试；流式活动聚合在 1000 条样本的部分场景下优于历史实现 | 只证明特定 CPU-only 路径的相对表现，**不是 HTTP+MySQL 吞吐** |
| 第一版原定 20 分钟测试 | 旧会话在约第 14–16 分钟出现 500/503 并中止；原始逐分钟记录缺失 | **错误演示**，不可视为完成 20 分钟 |
| 新增隔离 MySQL 20 分钟复测 | [Actions 原始运行](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39622692722)，Go 1.23.2 + MySQL 8.4，合成健康数据并发写入与规则扫描；**最终失败** | 已取得真实应用层与数据库证据；**未通过长时间稳定性验收** |
| 后续本地 20 分钟双库混合复测 | [2026-10-22 报告](docs/performance/2026-10-22-backend-hybrid/README.md)，旧 `b6aba05` 基线加修复；9,484 批、1,896,800 条采样，HTTP 500/503 均为 0，整机平均 CPU 98.41% | 仅证明该隔离负载；分析队列仍积压 833 条，整机 CPU 包含额外计算进程，当前合并源码未重跑此压测 |
| 冷热统计读取基准 | 同一批 10,000 条真实 MySQL 样本，统计处理均值 23.252 ms → 0.089078 ms，约 261 倍 | 仅限相同修订号的重复缓存命中；数据库自身已预热，不是物理冷盘、真实网络或上传容量 |

### 20 分钟 MySQL 复测：失败也必须保留

该运行日志记录 **1,987 次请求、1,959 次业务确认、28 次 HTTP 500、0 次 HTTP 503、19 次规则评估错误**；已确认 **391,800** 条合成健康数据写入，与 `health_sample` 和 `health_change` 的最终行数相等。错误集中出现在测试后段；**记录完整性不等于整个服务稳定性**。这是隔离的单进程 `httptest` + 真实 MySQL 负载，不覆盖多应用节点、真实设备、外部通知和生产网络。

**重要的 CI 陷阱：**最初的耐久工作流把 `go test` 通过管道传给 `tee`，未启用 `pipefail`，因此即便 `go test` 实际失败，Actions 步骤也可能显示绿色。随后已启用 `pipefail`；看测试原始日志与数据库核对结果，不能只看工作流结论。

**第二轮独立复测也未通过：**[修复退出码后的运行 #39629502257](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39629502257) 正确标记为失败，原始日志记录 **44 次请求错误、19 次规则评估错误**；**387,200** 条已确认样本与两张持久化表的最终行数吻合。第 20 分钟检查点累计出现 40 次 HTTP 500、0 次 HTTP 503；检查点之后仍有错误，不能仅凭检查点代替最终状态码分布。两轮数据不可简单比较为性能提升或回退。

- [第一轮耐久运行原始日志与 Artifact](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39622692722) 和 [第二轮失败运行与 Artifact](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39629502257)。
- [第一版测试错误演示](docs/performance/2026-10-21-go123/FIRST_20MIN_INCORRECT_DEMONSTRATION.md) 与 [机器可读证据状态](docs/performance/2026-10-21-go123/first_20min_incorrect_evidence.json)。
- [CPU/GC 性能目录](docs/performance/2026-10-21-go123/README.md) 与 [数据库 500/503 排查](docs/MYSQL_503_500_CONTINUATION_20261021.md)。

### 后续本地修复与冷热缓存：单独记录结果

[2026-10-22 完整报告](docs/performance/2026-10-22-backend-hybrid/README.md) 保留上述失败，并记录独立探测连接、离线扫描分页、0008/0009 基线索引后的本地 20 分钟复测。9,484 批成功确认对应两张表各新增 1,896,800 行，约 1,574.4 条采样/秒；写入 p99 仍约 14.8 秒，不能据此承诺生产延迟。PG 直接查询无错误，但排空后的分析审计为已投影 2,763 条、待处理 833 条，并非同步全部完成。

随后增加的有界内存缓存保留 MySQL 全量磁盘历史，每次命中仍核验权限和已提交修订号。约 261 倍的结果只针对重复统计处理，不能换算成上传或样本预处理吞吐。上述长压测及缓存基准均来自旧 `b6aba05` 基线加补丁；与更新后的 `preview` 合并另作定向验证，未重新运行 20 分钟，完整测试集也不能据此声称全绿。

### 开发者最小验证

在 `智慧养老手表管理系统/backend` 下，先准备 Go 1.23+ 与**专用测试库**：

```sh
go test -count=1 ./...
go vet ./...
# 实例必须与生产隔离；MySQL 用例会创建和删除随机测试数据库
TEST_MYSQL_DSN='专用测试 MySQL DSN' \
TEST_DATABASE_URL='专用测试 PostgreSQL URL' \
go test -race -count=1 -timeout 600s ./...
```

缺少相应 DSN 时，部分数据库集成用例会跳过；`go test` 返回成功不代表这些被跳过的用例经过验证。**20 分钟耐久测试采用显式 opt-in**，见 [工作流](.github/workflows/mysql-endurance.yml) 与 [中止/复测说明](docs/performance/2026-10-21-go123/INTERRUPTED-MYSQL-SOAK.md)，绝不能在生产库上运行。

## 界面与验证

![历史 2.0.2 登录界面](docs/screenshots/tauri-2.0.2-offline.png)

![历史 2.0.2 管理工作区](docs/screenshots/tauri-2.0.2-main.png)

以上为历史 2.0.2 实际 Tauri 截图，使用临时数据，未作为 2.0.6 界面验收图。当前界面保留山水标志、Source Serif 4 与思源宋体，支持大字、`F11` 全屏及 `Escape` 退出。

- 双库开发阶段：独立 MySQL 8.0.46 / PostgreSQL 16.4 上的 Go 全套竞争检测、11 组模拟 API 浏览器验收，以及 PG 停机时 15 项真实业务检查；恢复后核对队列归零与分析视图。
- 2.0.6 本地打包阶段：前端构建、Go 测试与静态检查、Tauri / NSIS 构建、应用和安装包版本、压缩包及 SHA-256 校验、新内置服务启动检查通过；打包阶段未重跑真实双库集成。**公开 v2.0.6 Release 仅上传发布快照源码及校验附件**。
- 本地 2.0.6 安装包与免安装版未签名，未实际执行安装器或验收原生桌面窗口。历史安装验收不代表新版已经安装验收。
- 当前 Go CI 已在 **MySQL 8.4** 测试容器中执行数据库集成测试；历史两轮 20 分钟 CI 负载失败，后续本地修复版的成功结果与当前合并源码的验证边界见 [最新报告](docs/performance/2026-10-22-backend-hybrid/README.md)。**真实手表、短信网关、生产迁移、远程 TLS、真实网络故障与长期生产可用性**仍未完成端到端验收。旧 2.0.6 桌面安装验收边界不因此改变。

版本验证和升级边界集中记录在 [2.0.6 改进说明](docs/releases/2.0.6.md)。本次整合保留远程健康采样、增量游标与统计接口，并适配 MySQL；测试证据见 [整合记录](docs/INTEGRATION_2.0.6.md)。此前本地安装包不包含此次健康采样整合，应从当前源码重建。

## 常见问题

| 现象 | 首先核对 |
| --- | --- |
| 软件能打开，但无法登录或保存 | MySQL 是否可用、DSN 是否指定业务库、迁移是否完成；网络在线不代表数据库就绪 |
| PG 断线，业务仍能保存 | 这是双库设计；查看分析积压与磁盘容量，恢复后补齐，不应切换业务主库 |
| 升级后看不到旧资料 | 当前 MySQL 与旧 PG 是否完成离线导入，图片是否另行复制；不要直接覆盖唯一备份 |
| 设备上报返回绑定版本冲突 | 核对采集时的绑定版本；不能把旧缓存换成新版本后归给新长者 |
| 已发送通知，任务仍未关闭 | 发送、送达、接单、处置和独立复核是不同状态 |
| HTTP 500/503 或数据库状态异常 | 查看 [500/503 排查记录](docs/MYSQL_503_500_CONTINUATION_20261021.md)、业务池等待与 SQL 慢查询；单个 503 不等于整库宕机，且健康探测与业务成功需分开监控 |
| 免安装版不能启动 | 是否完整解压、两个 EXE 是否同目录、WebView2 是否存在；先确认附件真实版本 |
| 下载程序与源码版本不一致 | 从同一 2.1.0 Release 获取程序和源码，并核对 SHA-256 |

反馈时提供版本或提交、文件名、系统、数据库就绪状态和脱敏复现步骤即可，详见 [支持说明](docs/SUPPORT.md)。

## 文档与参与

| 目的 | 入口 |
| --- | --- |
| 源码下载与历史附件 | [附件索引](docs/DOWNLOADS.md) · [About 文案](docs/ABOUT.md) |
| 当前配置、双库与迁移 | [双库指南](docs/DUAL_DATABASES.md) · [使用说明](docs/SETUP.md) |
| 架构与照护闭环 | [架构总览](docs/architecture/overview.md) · [实现协议](docs/ARCHITECTURE.md) · [照护部署](docs/CARE_SERVICE_DEPLOYMENT.md) |
| 开发与验证 | [工程说明](智慧养老手表管理系统/README.md) · [前端](智慧养老手表管理系统/oldIot/README.md) · [后端](智慧养老手表管理系统/backend/README.md) · [测试分层](docs/TESTING.md) · [最新性能证据](docs/performance/2026-10-22-backend-hybrid/README.md) · [历史性能证据](docs/performance/2026-10-21-go123/README.md) |
| 备份与文件职责 | [备份恢复](docs/runbooks/backup-restore.md) · [逐文件索引](docs/FILES.md) |
| 贡献、支持与历史 | [贡献指南](CONTRIBUTING.md) · [支持说明](docs/SUPPORT.md) · [更新日志](CHANGELOG.md) · [历史决定](docs/decisions/README.md) |

2.1.0 从 `preview` 发布；不修改 `main` 或移动旧标签。历史 ADR 和验收文档保留其版本上下文；当前配置以本 README、2.0.6 说明和双库指南为准。

欢迎通过 [Issues](https://github.com/WatchTestCs/smart-elderly-watch-management/issues) 反馈可复现问题，安全问题按 [安全策略](.github/SECURITY.md) 私下报告。请勿公开真实人员数据、数据库连接串或密钥。[MIT License](LICENSE)；字体与第三方组件遵循各自许可。
