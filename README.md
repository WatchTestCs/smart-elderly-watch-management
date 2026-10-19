<p align="center"><img src="智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-192.webp" width="88" height="88" alt="岁宁标志"></p>

# 岁宁 · 智慧养老管理

**简体中文** | [English](README.en.md)

[![License: MIT](https://img.shields.io/badge/license-MIT-235442)](LICENSE) [![Preview: 2.0.6](https://img.shields.io/badge/preview-2.0.6-9b7a40)](docs/releases/2.0.6.md)

**岁月有伴，日常安宁。**

岁宁面向照护机构与家属，提供长者档案、手表设备、健康与活动记录、位置告警、照护任务和公告互动。当前 2.0.6 采用 **Tauri 2 / Rust 桌面 + Vue 3 / Vite 界面 + Go 业务服务 + MySQL 主库 + PostgreSQL 分析库**，数据库由部署方独立管理。

[源码与历史附件](docs/DOWNLOADS.md) · [2.0.6 改进与升级](docs/releases/2.0.6.md) · [系统架构](#系统架构) · [开始使用](#开始使用) · [双库部署与迁移](docs/DUAL_DATABASES.md) · [源码工程](智慧养老手表管理系统/README.md) · [更新日志](CHANGELOG.md)

## 仓库目录与根文件

| 路径 | 用途 |
| --- | --- |
| [.github/](.github/) | 漏洞披露、依赖更新、PR 模板和自动检查工作流。 |
| [deploy/](deploy/) | MySQL／PG 部署模板与独立服务运维探测脚本；不包含应用容器。 |
| [docs/](docs/) | 使用、架构、迁移、照护运维、历史版本与[逐文件职责索引](docs/FILES.md)。 |
| [tests/ui/](tests/ui/) | 无外部依赖的浏览器可访问性冒烟检查。 |
| [智慧养老手表管理系统/](智慧养老手表管理系统/README.md) | 可构建的桌面工程：Tauri、Go 后端、Vue 前端和手表模拟器。 |
| [.gitignore](.gitignore) | 排除依赖缓存、数据库、构建产物与私钥。 |
| [CHANGELOG.md](CHANGELOG.md) | 按版本记录功能变化、修复与兼容性。 |
| [CONTRIBUTING.md](CONTRIBUTING.md) | 贡献流程、分支和验证约定。 |
| [LICENSE](LICENSE) | 仓库 MIT 许可和版权声明。 |
| [README.en.md](README.en.md) | 英文项目介绍、技术栈、下载和部署入口。 |
| [README.md](README.md) | 中文项目首页与当前源码入口。 |
| [architecture.md](architecture.md) | 根目录的系统组件和数据流概览。 |

## 2.0.6 与交付状态

2.0.6 已完成本地 Windows x64 安装版、免安装版和源码包构建；本次交付未发布到 GitHub Releases。文件名分别为 `Suining-Setup-2.0.6-win-x64.exe`、`Suining-Portable-2.0.6-win-x64.zip` 和 `Suining-Source-2.0.6.zip`，以随交付的 `SHA256SUMS.txt` 核验。

- **界面补齐：**导航仅展开当前分组，统一术语，长者档案窄屏卡片化，五个管理页面补齐加载与失败重试，调整宽屏栅格和统计口径。
- **资料与设备：**先建档再绑定手表，档案编辑不改写设备归属，保留自定义阈值，修正生活资料字段，缺测指标保留 NULL。
- **照护闭环：**SOS 接单、处置、另一名主管独立复核；任务和履历分页，关闭关联告警并刷新统计。
- **健康采样：**保留远程批量采样、版本修订、绑定隔离、增量同步与单来源统计，改用 MySQL 持久化；旧前端和旧上报路径仍保持各自行为，见 [接口说明](智慧养老手表管理系统/backend/docs/health-data.md)。
- **双库分工：**MySQL 承担全部核心业务，PG 异步接收分析投影；附旧 PG 离线导入和分析回填工具。

### 获取源码与选择附件

截至 2026-10-19，当前 `preview` 为 **2.0.6 双库整合源码**，最新公开源码附件为 **2.0.5-optimized.1**，最新公开 Windows 安装／免安装附件仍为 **2.0.2**。版本号和交付类型需要同时核对，不能把旧安装包套用当前数据库配置。

| 需要什么 | 下载入口 | 使用范围 |
| --- | --- | --- |
| 当前 2.0.6 源码 | [preview 源码 ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/archive/refs/heads/preview.zip) · [整合代码固定提交](https://github.com/WatchTestCs/smart-elderly-watch-management/archive/3a889f3aa24986720418d8ceba890754c07cdcd0.zip) | MySQL 主库＋PG 分析；自行构建，分支 ZIP 随提交变化 |
| 最近公开的旧版源码 | [2.0.5 架构优化版](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.5-optimized.1) | PG 单库历史源码，不含本次双库与健康采样整合 |
| 历史 Windows 软件 | [2.0.2 安装与免安装](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.2-preview.1) | 旧 PG 单库程序，不是 2.0.6 |
| 历史 Linux 附件 | [Linux 软件与数据库工具说明](docs/LINUX.md) | 独立历史附件，未证明与当前源码兼容 |

所有历史版本、具体文件、校验入口与兼容性见 [下载与历史附件索引](docs/DOWNLOADS.md)。GitHub 自动生成的源码 ZIP 不等于 NSIS 安装器，也不等于经过本地打包验证的 Release 附件。

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

Vue 通过 HTTP 使用 Go API，不直接连接数据库。首次管理员初始化通过受限 Tauri IPC，由 Rust 附加桌面令牌；网页不持有该令牌或数据库连接串。普通业务写入使用会话、角色和 CSRF 校验，设备接口使用独立设备鉴权。会话保存在 Go 内存，服务重启后需重新登录。

### 双库数据流与故障行为

| 存储 | 保存内容 | 恢复与一致性 |
| --- | --- | --- |
| MySQL 主库 | 账号、人员与设备、健康原始记录、轨迹、告警、照护、审计、公告评论、通知及分析 outbox | 唯一业务数据源；业务 API 始终读写 MySQL |
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

1. 先从当前源码构建匹配的 2.0.6 Windows 程序，历史公开安装包不包含本次整合。安装构建所得的 Setup，或完整解压相应免安装包后运行 `suining.exe`，保留同目录 `suining-server.exe`。最终用户无需安装 Go、Rust 或 Node.js。Setup 在缺少 WebView2 时联网补装，免安装版需要系统已有 WebView2。
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

Windows 构建需要 PowerShell 7、Node.js（`^20.19.0 || >=22.12.0`）、Go、Rust 和 MSVC / Windows SDK。Go 模块声明最低 1.24.0，2.0.6 本地构建使用 Go 1.27.2；Rust 包声明最低 1.85。从工程目录执行：

```powershell
cd 智慧养老手表管理系统
./desktop/validate-version.ps1
./desktop/build.ps1
```

构建顺序为 **Vue 生产资源 → Go 内嵌资源、测试与编译 → sidecar SHA-256 → Tauri 原生程序 → NSIS 安装包**。脚本支持 `CARGO_TARGET_DIR`，产物在工程目录 `outputs/v2.0.6-preview`。默认构建不签名；可选 `-SelfSign` 仅用于本机自签，不能提供公开受信任签名。免安装 ZIP、源码 ZIP 和交付说明属于单独整理步骤。开发代理、真实数据库测试要求见 [工程 README](智慧养老手表管理系统/README.md) 和 [后端 README](智慧养老手表管理系统/backend/README.md)。

## 界面与验证

![历史 2.0.2 登录界面](docs/screenshots/tauri-2.0.2-offline.png)

![历史 2.0.2 管理工作区](docs/screenshots/tauri-2.0.2-main.png)

以上为历史 2.0.2 实际 Tauri 截图，使用临时数据，未作为 2.0.6 界面验收图。当前界面保留山水标志、Source Serif 4 与思源宋体，支持大字、`F11` 全屏及 `Escape` 退出。

- 双库开发阶段：独立 MySQL 8.0.46 / PostgreSQL 16.4 上的 Go 全套竞争检测、11 组模拟 API 浏览器验收，以及 PG 停机时 15 项真实业务检查；恢复后核对队列归零与分析视图。
- 2.0.6 打包阶段：前端构建、Go 测试与静态检查、Tauri / NSIS 构建、应用和安装包版本、压缩包及 SHA-256 校验、新内置服务启动检查通过；打包阶段未重跑真实双库集成。
- 本地 2.0.6 安装包与免安装版未签名，未实际执行安装器或验收原生桌面窗口。历史安装验收不代表新版已经安装验收。
- 未验证真实手表、短信网关、生产迁移、远程 TLS、长期负载或 MySQL 8.4 容器运行。Compose 已做静态校验，不能视为容器实测。浏览器模拟与协议模拟也不能替代真实业务或硬件联调。

版本验证和升级边界集中记录在 [2.0.6 改进说明](docs/releases/2.0.6.md)。本次整合保留远程健康采样、增量游标与统计接口，并适配 MySQL；测试证据见 [整合记录](docs/INTEGRATION_2.0.6.md)。此前本地安装包不包含此次健康采样整合，应从当前源码重建。

## 常见问题

| 现象 | 首先核对 |
| --- | --- |
| 软件能打开，但无法登录或保存 | MySQL 是否可用、DSN 是否指定业务库、迁移是否完成；网络在线不代表数据库就绪 |
| PG 断线，业务仍能保存 | 这是双库设计；查看分析积压与磁盘容量，恢复后补齐，不应切换业务主库 |
| 升级后看不到旧资料 | 当前 MySQL 与旧 PG 是否完成离线导入，图片是否另行复制；不要直接覆盖唯一备份 |
| 设备上报返回绑定版本冲突 | 核对采集时的绑定版本；不能把旧缓存换成新版本后归给新长者 |
| 已发送通知，任务仍未关闭 | 发送、送达、接单、处置和独立复核是不同状态 |
| 免安装版不能启动 | 是否完整解压、两个 EXE 是否同目录、WebView2 是否存在；先确认附件真实版本 |
| 源码写着 2.0.6，下载程序却是 2.0.2 | 分支源码与历史二进制分开交付，按 [附件索引](docs/DOWNLOADS.md) 选择 |

反馈时提供版本或提交、文件名、系统、数据库就绪状态和脱敏复现步骤即可，详见 [支持说明](docs/SUPPORT.md)。

## 文档与参与

| 目的 | 入口 |
| --- | --- |
| 源码下载与历史附件 | [附件索引](docs/DOWNLOADS.md) · [About 文案](docs/ABOUT.md) |
| 当前配置、双库与迁移 | [双库指南](docs/DUAL_DATABASES.md) · [使用说明](docs/SETUP.md) |
| 架构与照护闭环 | [架构总览](architecture.md) · [实现协议](docs/ARCHITECTURE.md) · [照护部署](docs/CARE_SERVICE_DEPLOYMENT.md) |
| 开发与验证 | [工程说明](智慧养老手表管理系统/README.md) · [前端](智慧养老手表管理系统/oldIot/README.md) · [后端](智慧养老手表管理系统/backend/README.md) · [测试分层](docs/TESTING.md) |
| 备份与文件职责 | [备份恢复](docs/runbooks/backup-restore.md) · [逐文件索引](docs/FILES.md) |
| 贡献、支持与历史 | [贡献指南](CONTRIBUTING.md) · [支持说明](docs/SUPPORT.md) · [更新日志](CHANGELOG.md) · [历史决定](docs/decisions/README.md) |

当前工作只更新 `preview`；不修改 `main`、移动旧标签或随文档更新发布附件。历史 ADR 和验收文档保留其版本上下文；当前配置以本 README、2.0.6 说明和双库指南为准。

欢迎通过 [Issues](https://github.com/WatchTestCs/smart-elderly-watch-management/issues) 反馈可复现问题，安全问题按 [安全策略](.github/SECURITY.md) 私下报告。请勿公开真实人员数据、数据库连接串或密钥。[MIT License](LICENSE)；字体与第三方组件遵循各自许可。
