<p align="center">
  <img src="智慧养老手表管理系统/frontend/public/static/brand/suining-logo-192.webp" width="88" height="88" alt="岁宁·智慧养老">
</p>

<h1 align="center">岁宁 · 智慧养老管理</h1>

<p align="center">岁月有伴，日常安宁。<br>连接长者、家属与照护人员的开源手表管理平台。</p>

<p align="center">
  <a href="README.md">简体中文</a> · <a href="README.en.md">English</a>
</p>

[![Preview CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml/badge.svg?branch=preview)](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml)

[下载](#下载) · [主要功能](#主要功能) · [开始使用](#开始使用) · [系统架构](#系统架构) · [源码与构建](#源码与构建) · [验证与性能](#验证与性能) · [文档导航](#文档导航)

岁宁面向照护机构与家属，在同一工作台内集中管理长者档案、手表设备、健康记录、位置告警与照护任务。客户端与服务端基于 Tauri 2、Rust、Vue 3、Vite 与 Go 构建，使用独立部署的 MySQL 保存业务数据，并通过 PostgreSQL 异步处理统计分析。

仓库根目录下的 [SuiningWatch](SuiningWatch/README.md) 是配套的 HarmonyOS 手表端工程，用于模拟器联网调试。其构建方式与测试边界见[手表端整合记录](docs/WATCH_INTEGRATION.md)，当前仅生成模拟数据，未接入真实硬件传感器，也未配置真机发布签名。

![岁宁管理员工作台：设备异常、位置告警与平台公告](docs/screenshots/overview-desktop.png)

*上图为使用测试数据的界面示例，仅供预览布局。*

## 下载

2.1.0 · Windows x64（预览版）

| 版本类型 | 下载链接 | 适用场景 |
| --- | --- | --- |
| 安装版 | [Windows Setup](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Setup-2.1.0-win-x64.exe) | 独立安装程序，适合首次部署 |
| 免安装版 | [Portable ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Portable-2.1.0-win-x64.zip) | 解压即用，完整解压后运行，需已有 WebView2 |
| 发布源码 | [Source ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Source-2.1.0.zip) | 与标签版本完全一致，包含数据库迁移脚本 |

校验与发布信息：[SHA-256](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/SHA256SUMS.txt) · [完整发布记录](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.1.0)

安装包不含数据库服务，运行客户端无需安装 Go、Rust 或 Node.js。当前 Windows 安装包与执行文件未做数字签名。

> [!NOTE]
> v2.1.0 发行附件基于固定提交 [e3cf064](https://github.com/WatchTestCs/smart-elderly-watch-management/commit/e3cf064) 构建。如需体验后续提交的改动，请按下文说明从源码构建。

## 主要功能

| 模块 | 功能范围 |
| --- | --- |
| 长者与家庭 | 管理长者、监护人和家属档案；校验有效监护人关联及监护人的 0–150 岁整数年龄，保护仍被长者引用的监护人；生成家属注册邀请码 |
| 手表设备 | 登记与绑定设备、校验设备密钥、查看在线状态、设置监测阈值 |
| 健康与日常 | 查询心率、单项血压、睡眠、步数和日常生活记录 |
| 位置与告警 | 查看最新上报位置、异常记录、SOS 求助与历史事件 |
| 照护协作 | 接单、开始处置、记录措施，并由另一名主管独立复核结案 |
| 公告与互动 | 发布图文公告、管理评论，区分管理端与家属端视图 |
| 桌面无障碍 | 支持大字模式、全键盘操作、`F11` 全屏及窄屏自适应布局 |

日常业务按以下顺序流转：建立长者档案，登记并绑定手表，接收健康与位置上报，触发告警并生成工单，照护人员接单处置，最后由另一名主管独立复核。

系统按角色隔离权限：家属只能查看已关联长者的记录；主管拥有完整后台管理权限；护理员与审计员仅限访问照护协作页面及个人双因素认证（2FA）设置。

## 开始使用

### Windows 桌面端

1. 运行 Setup 安装程序，或将免安装 ZIP 完整解压到本地目录。
2. 准备独立的 MySQL 业务数据库，需要使用异步分析时再额外部署 PostgreSQL。可以直接使用仓库提供的[双库 Compose 模板](deploy/dual-db.compose.yml)，其 MySQL 与 PostgreSQL 默认端口分别为 13306 和 15432。
3. 在 PowerShell 中配置数据库连接与会话加密密钥，随后启动客户端：

```powershell
$env:SUINING_MYSQL_DSN='suining:替换密码@tcp(127.0.0.1:13306)/suining'
$env:SUINING_PG_ANALYTICS_URL='postgres://suining_analytics:替换密码@127.0.0.1:15432/suining_analytics?sslmode=disable'
$env:SESSION_BACKEND='database'
$env:DATA_ENCRYPTION_KEY='替换为已妥善保存的32字节随机密钥的Base64'
& '.\suining.exe'
```

上述配置仅适用于受控的本机环境，跨主机连接数据库时请开启并验证 TLS。如果不配置 PostgreSQL，核心业务依然正常运转，但需留意 MySQL 中待同步分析事件的积压情况。首次部署时请生成 `DATA_ENCRYPTION_KEY` 并妥善备份，切勿在后续启动时随意更换。

4. MySQL 自动迁移完成后，按桌面引导创建首位管理员账号。系统不设默认手机号或初始通用密码，家属需凭管理员生成的邀请码注册。
5. 使用时先建立长者档案，再登记并绑定手表。本地图片与日志默认保存在 `%APPDATA%\岁宁`，日常运维时请将图片目录与数据库分别备份。

修改环境变量后需重启客户端生效。直接运行 Go 服务端时读取 `MYSQL_DSN` 与 `PG_ANALYTICS_URL`，通过桌面端启动时则读取带 `SUINING_` 前缀的同名变量。旧版 `DATABASE_URL` 与 `SUINING_DATABASE_URL` 已弃用。

### 本地容器化平台

仓库提供了包含 MySQL 8.4、PostgreSQL 16、Go 后端、Nginx 与 Prometheus 的[完整 Compose 编排文件](deploy/platform.compose.yml)。配置好环境变量后，可通过脚本一键启动：

```sh
python3 scripts/start-platform.py
# 待服务就绪后，按提示创建首位管理员
python3 scripts/bootstrap-admin.py
```

启动后应用默认监听 `http://127.0.0.1:8080`，Prometheus 指标页位于 `http://127.0.0.1:9090`。启动脚本首次运行时会生成 `deploy/secrets/` 与 `deploy/platform.env`，保留已有密钥和密码；不会生成 TLS 证书。MySQL 业务服务先启动，随后尝试启动 PostgreSQL，其失败不阻塞业务。

### 旧库迁移与长期值守

如果从早期使用 PostgreSQL 作为业务主库的版本升级，请先停止旧服务写入并备份数据库与图片，然后使用 `transfer-postgres` 工具完成预检查，将数据导入全新的空 MySQL 库，最后手动复制图片目录。安装程序不会自动迁移旧库，且全量导入过程不可逆。

桌面客户端默认监听本机回环动态端口，关闭窗口即终止本地 Go 服务。如果需要全天候接收真实手表上报并处理紧急求助，请独立部署常驻 Go 服务、配置受控公网或局域网入口、接入外部通知网关，并安排值班人员跟进。

## 系统架构

```mermaid
flowchart LR
    R["Tauri / Rust"] -->|"校验、启动与退出"| G["Go API 与后台作业"]
    V["Vue 3 / WebView2"] -->|"同源 HTTP"| G
    V -->|"受限初始化 IPC"| R
    G -->|"业务事务"| M[("MySQL 主库")]
    M -->|"事务内写入"| O["分析 outbox"]
    O -->|"Go 异步投影 / 幂等重试"| P[("PostgreSQL 分析库")]
    G --> F["本地图片与日志"]
```

MySQL 是唯一的业务数据源，PostgreSQL 仅存放可随时重建的分析投影。前端 Vue 应用通过同源 HTTP 访问 Go 接口，不直接连接数据库，也不接触桌面初始化令牌或数据库凭据。

| 运行异常或状态 | 系统处理方式 |
| --- | --- |
| MySQL 未连接或迁移未完成 | 桌面界面正常打开，业务 API 返回 `503 DATABASE_UNAVAILABLE` |
| PostgreSQL 未配置或离线 | 核心业务正常读写，已提交的分析事件暂存于 MySQL outbox，待恢复后自动补齐 |
| 用户提交保存失败 | 页面保留当前已填内容并提示手动重试，不自动重放写请求 |
| 会话配置缺失或无效 | 状态接口返回 `configError`，提示修正环境变量后重启 |

`GET /healthz` 仅用于探测 HTTP 进程是否存活，数据库连接与迁移状态由 `GET /api/v1/service/status` 返回。完整的进程交互与鉴权细节见[架构说明](docs/ARCHITECTURE.md)，健康采样的跨窗口步数分摊、日界切分与阈值判定规则见[健康数据接口约定](智慧养老手表管理系统/backend/docs/health-data.md#跨窗口步数与夜间累计计数)。

## 源码与构建

日常开发基于 `preview` 分支；如需复现已发布的安装包，请直接下载上方的发行版源码包。

```sh
git clone --branch preview https://github.com/WatchTestCs/smart-elderly-watch-management.git
cd smart-elderly-watch-management
```

```text
智慧养老手表管理系统/
├── desktop/     Tauri / Rust 桌面壳、原生权限与 Windows 打包脚本
├── backend/     Go API、数据库迁移、照护工单与分析投影作业
├── frontend/    Vue 3 / Vite 前端界面与 Playwright 浏览器测试
└── watch-sim/   手表上报协议模拟器
SuiningWatch/    HarmonyOS (ArkTS) 穿戴端模拟工程
deploy/          Docker Compose、数据库与监控告警配置
scripts/         平台启动、OpenAPI 生成与基准测试脚本
tests/           UI 冒烟测试
docs/            架构设计、部署手册、版本说明与验证记录
evidence/        自动化测试与故障演练留档数据
```

在 Windows 上执行完整桌面构建，需要安装 PowerShell 7、Node.js（`^20.19.0 || >=22.12.0`）、Go（`go.mod` 要求 `1.26.0+`）、Rust 1.85+、MSVC 生成工具与 Windows SDK。具体环境准备见[构建环境说明](docs/SETUP.md)。

```powershell
cd 智慧养老手表管理系统
./desktop/validate-version.ps1
./desktop/build.ps1
```

构建脚本依次编译 Vue 静态资源、内嵌静态资源的 Go 服务端、sidecar SHA-256 校验文件、Tauri 原生壳以及 NSIS 安装包，最终产物输出到 `outputs/v2.1.0-preview`。

体征异常处置会拒绝无效的 `kind`，避免误清除心率和血压两类异常；按记录处置时，回显原因对应被处理的那条记录。详见[处置修复与验证范围](docs/patch-review/2026-10-31-vitals-resolution.md)。

## 验证与性能

| 验证维度 | 记录与说明 |
| --- | --- |
| 分支自动化检查 | [![Preview CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml/badge.svg?branch=preview)](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml) |
| 2.1.0 发布验证 | [2.1.0 版本说明](docs/releases/2.1.0.md)与随包 `release-manifest.json` |
| 请求边界与生命周期 | [2026-10-26 边界重构记录](docs/refactoring/2026-10-26-boundaries/README.md)：涵盖会话取消、统一权限表、JWT、字段脱敏与契约对齐 |
| 睡眠分段统计 | [2026-10-26 睡眠统计优化](docs/performance/2026-10-26-sleep/README.md)：统一时间排序与阶段累计，覆盖乱序、跨窗口与重复样本测试 |
| 算法与前端读取调度 | [2026-10-26 调度与算法迭代](docs/performance/2026-10-26-algorithms/README.md)：包含统计内存复用、限流分片、指令查找与分页优化 |
| 后端混合负载与缓存 | [2026-10-22 混合压测报告](docs/performance/2026-10-22-backend-hybrid/README.md)：记录隔离环境下的延迟与内存基线 |
| 耐久测试与 GC 实验 | [Go 1.23 耐久测试记录](docs/performance/2026-10-21-go123/README.md)与 [Go 运行时调优记录](docs/performance/2026-10-21-go123/README.md) |

在 `智慧养老手表管理系统/backend` 目录下可执行基础单元与静态检查：

```sh
go test -count=1 ./...
go vet ./...
```

如需运行完整数据库集成与竞态测试，请先配置专用的测试库环境变量 `TEST_MYSQL_DSN` 与 `TEST_DATABASE_URL`，再执行 `go test -race -count=1 -timeout 600s ./...`（未配置时数据库集成用例会自动跳过）。

仓库内的压测与自动化用例均在本地或容器隔离环境中完成。正式投入生产前，仍需针对真实穿戴硬件、外部短信或语音网关、生产数据迁移、跨网 TLS 证书及值班响应流程开展独立验收。同时请注意，无新增异常上报仅代表未收到新数据，通知发送成功也仅代表网关已受理，实际照护结果以人工跟进与复核记录为准。

## 常见问题

| 现象 | 排查方向 |
| --- | --- |
| 界面能打开，但无法登录或保存 | 检查 MySQL 是否启动、库名是否正确、迁移是否完成，以及 `SESSION_BACKEND` 和 `DATA_ENCRYPTION_KEY` 是否有效 |
| PostgreSQL 离线，但日常操作仍能保存 | 属于预期设计；业务直接写入 MySQL，留意磁盘空间与分析事件积压即可 |
| 升级版本后找不到历史记录 | 确认是否已执行离线迁移工具导入旧库数据，并手动复制了旧版图片目录 |
| 外部手表无法连接桌面端服务 | 桌面端默认仅监听本机回环地址，外部设备接入需独立部署服务端并开放受控网络端口 |
| 免安装版双击无法启动 | 确认已完整解压 ZIP、主程序与 Go sidecar 位于同一目录、系统已安装 WebView2，并核对 SHA-256 |

## 文档导航

| 主题 | 相关文档 |
| --- | --- |
| 下载、版本与升级 | [下载索引](docs/DOWNLOADS.md) · [2.1.0 发布说明](docs/releases/2.1.0.md) · [更新日志](CHANGELOG.md) |
| 数据库、迁移与备份 | [双库配置指南](docs/DUAL_DATABASES.md) · [备份与恢复手册](docs/runbooks/backup-restore.md) |
| 设备接入与照护运维 | [照护服务部署](docs/CARE_SERVICE_DEPLOYMENT.md) · [支持与故障排查](docs/SUPPORT.md) |
| 源码开发与模块说明 | [工程总览](智慧养老手表管理系统/README.md) · [前端说明](智慧养老手表管理系统/frontend/README.md) · [后端说明](智慧养老手表管理系统/backend/README.md) · [手表端说明](SuiningWatch/README.md) |
| 架构设计与测试 | [架构总览](docs/architecture/overview.md) · [架构决策记录 (ADR)](docs/decisions/README.md) · [测试指南](docs/TESTING.md) |

提交 [Issue](https://github.com/WatchTestCs/smart-elderly-watch-management/issues) 反馈问题时，请附上版本号或 Git 提交哈希、操作系统环境、网络拓扑结构以及脱敏后的应用日志。

本项目采用 [MIT 许可证](LICENSE) 开源，仓库包含的字体、地图及第三方依赖遵循各自原有许可证。

> **接口不兼容变更提醒：** 当前 `preview` 分支仅暴露 `/api/v1/` 路由，已移除无版本号的 `/api/` 别名；浏览器端会话 Cookie 统一更名为 `SUINING_SESSION`。
