<p align="center">
  <img src="智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-192.webp" width="88" height="88" alt="岁宁标志">
</p>

<h1 align="center">岁宁 · 智慧养老管理</h1>

<p align="center"><strong>岁月有伴，日常安宁。</strong><br>连接长者、家属与照护人员的开源手表管理平台。</p>

<p align="center">
  <a href="README.md">简体中文</a> · <a href="README.en.md">English</a>
</p>

[![Preview CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml/badge.svg?branch=preview)](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml?query=branch%3Apreview) [![Preview: 2.1.0](https://img.shields.io/badge/preview-2.1.0-9b7a40)](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.1.0) [![Go: 1.23+](https://img.shields.io/badge/Go-1.23%2B-00ADD8)](智慧养老手表管理系统/backend/go.mod) [![License: MIT](https://img.shields.io/badge/license-MIT-235442)](LICENSE)

[下载安装](#下载) · [主要功能](#主要功能) · [开始使用](#开始使用) · [系统架构](#系统架构) · [源码与构建](#源码与构建) · [验证与性能](#验证与性能) · [文档导航](#文档导航)

岁宁面向照护机构与家属，将长者档案、手表设备、健康记录、位置告警与照护任务放在同一工作台。采用 **Tauri 2 / Rust + Vue 3 / Vite + Go**，以独立部署的 **MySQL** 保存业务数据，**PostgreSQL** 承接异步分析。

本分支同时包含 [HarmonyOS 手表端 SuiningWatch](SuiningWatch/README.md)，与网页和后端源码
一并管理。该原生 Wearable 工程用于模拟数据联调；构建与验收范围见
[手表端整合记录](docs/WATCH_INTEGRATION.md)，不包含真实传感器采集或真机签名发布。

![岁宁管理员工作台：设备异常、位置告警与平台公告](docs/screenshots/overview-desktop.png)

*仓库已有界面示例，使用测试数据；不作为 2.1.0 原生窗口验收记录。*

## 下载

**2.1.0 · Windows x64 · 预览渠道**

| 选择 | 下载 | 使用方式 |
| --- | --- | --- |
| 安装版 | [Windows Setup](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Setup-2.1.0-win-x64.exe) | 按安装向导操作；WebView2 缺失时联网补装 |
| 免安装版 | [Portable ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Portable-2.1.0-win-x64.zip) | 完整解压，运行 `suining.exe`，保留同目录 `suining-server.exe`；需已有 WebView2 |
| 发布版源码 | [Source ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Source-2.1.0.zip) | 与上述程序对应的固定源码快照 |

校验与来源：[SHA-256](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/SHA256SUMS.txt) · [构建清单](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/release-manifest.json) · [升级说明](docs/releases/2.1.0.md) · [历史版本](docs/DOWNLOADS.md)

安装包不包含数据库服务，最终用户无需安装 Go、Rust 或 Node.js。当前 Windows 包未签名。

> [!NOTE]
> **发布版与开发源码分别维护。** v2.1.0 附件固定于 [e3cf064](https://github.com/WatchTestCs/smart-elderly-watch-management/commit/e3cf06411ca99e83a4c982316327548d8d03160a)；`preview` 持续更新，已增加配置错误诊断和 SQL 翻译修复。下载分支源码不等于下载发布快照，现有 Release 附件未替换。

## 主要功能

| 场景 | 可以做什么 |
| --- | --- |
| 长者与家庭 | 管理长者、监护人和家属档案，通过邀请码关联家人 |
| 手表设备 | 登记、绑定、鉴权、查看在线状态与设置监测阈值 |
| 健康与日常 | 查询心率、单项血压、睡眠、步数和生活记录 |
| 位置与告警 | 查看最近上报位置、异常记录、SOS 与事件履历 |
| 照护协作 | 接单、开始处置、登记措施、由另一名主管独立复核 |
| 公告与互动 | 发布图文公告、管理评论，提供管理端与家属端视图 |
| 桌面操作 | 大字模式、键盘操作、`F11` 全屏及窄屏适配 |

典型流程：**建立档案 → 登记并绑定手表 → 接收记录 → 告警建单 → 人员处置 → 独立复核**。

家属仅访问关联家人的记录；主管保留现有后台权限，护理员与审计员只能进入实行细分岗位检查的照护接口以及本人双因素认证设置，不能访问其他传统管理员接口。未分配岗位的历史 ADMIN 仍按主管兼容处理，上线前必须清点并分配。地图使用 Leaflet 与 OpenStreetMap 瓦片，需要相应网络服务。

## 开始使用

### Windows 桌面

1. 下载并安装，或完整解压免安装包。
2. 准备独立 MySQL 业务库；需要分析时再配置 PostgreSQL。可使用[双库 Compose 模板](deploy/dual-db.compose.yml)，数据库端口默认仅映射回环地址。
3. 在启动软件的 PowerShell 中配置连接和会话密钥，然后运行程序：

```powershell
$env:SUINING_MYSQL_DSN='suining:替换密码@tcp(127.0.0.1:13306)/suining'
$env:SUINING_PG_ANALYTICS_URL='postgres://suining_analytics:替换密码@127.0.0.1:15432/suining_analytics?sslmode=disable'
$env:SESSION_BACKEND='database'
$env:DATA_ENCRYPTION_KEY='替换为已妥善保存的32字节随机密钥的Base64'
& '.\suining.exe'
```

上例仅用于受控本机连接；远程连接需验证 TLS。PG 分析可暂不配置，但需监控 MySQL 中积压的分析事件。`DATA_ENCRYPTION_KEY` 应在首次部署时生成并妥善备份，已有部署沿用原密钥，不能每次启动重新生成。完整配置见[后端说明](智慧养老手表管理系统/backend/README.md)。

4. MySQL 迁移完成后，通过桌面初始化首位管理员。**没有默认手机号或通用密码**；家属通过邀请码注册。
5. 先建立长者档案，再登记、绑定手表。数据库与图片分别备份，默认本地数据目录为 `%APPDATA%\岁宁`。

配置修改后重启。直接运行 Go 使用 `MYSQL_DSN` / `PG_ANALYTICS_URL`；桌面使用上述 `SUINING_` 连接变量。旧 `DATABASE_URL` / `SUINING_DATABASE_URL` 不作为主库回退配置。

### 本地开发平台

仓库提供包含 MySQL 8.4、PostgreSQL 16、Go、Nginx 与 Prometheus 的[完整 Compose 平台](deploy/platform.compose.yml)。准备 Docker Compose 与 Python 3，在仓库根目录运行：

```sh
python3 scripts/start-platform.py
# 服务就绪后，交互式创建首位管理员
python3 scripts/bootstrap-admin.py
```

默认应用地址为 `http://127.0.0.1:8080`，指标页面为 `http://127.0.0.1:9090`。启动脚本生成缺失的 `deploy/secrets/` 和 `deploy/platform.env`，保留已有文件；`--validate` 也会生成缺失配置，但只校验 Compose、不启动服务。密钥和环境文件不可提交到 Git。部署细节见[平台说明](docs/PLATFORM_UPGRADE_2.0.6.md)。

### 升级与持续值守

- **从旧 PG 业务库迁移：**备份数据库与图片，停止写入，再用 `transfer-postgres` 预检查并导入空 MySQL；图片单独复制。安装不会自动迁移旧库，Java/H2 不在该导入工具的支持范围内。见[迁移指南](docs/DUAL_DATABASES.md)。
- **设备接入与长期值守：**桌面默认监听回环动态端口，关闭窗口会结束本地 Go 服务。持续上报需独立常驻后台、受控网络入口、通知网关和人工响应流程。见[照护部署](docs/CARE_SERVICE_DEPLOYMENT.md)。

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

MySQL 是唯一业务数据源，PG 是可重建的分析投影。Vue 不直接连接数据库，也不持有桌面初始化令牌或数据库凭据。

| 情况 | 系统行为 |
| --- | --- |
| MySQL 不可用或迁移未完成 | 界面仍可打开，业务 API 返回 `503 DATABASE_UNAVAILABLE` |
| PG 未配置或不可用 | 核心业务继续，已提交分析事件保留在 MySQL，恢复后补齐 |
| 用户写入失败 | 保留输入，提示主动重试，不自动重放业务请求 |
| 会话配置无效（当前 `preview`） | 状态返回 `configError`，提示修正配置并重启 |

`/healthz` 仅表示 HTTP 进程存活；数据库就绪和分析连接状态由 `/restful/service/status` 报告。详细协议见[架构总览](docs/architecture/overview.md)、[实现架构](docs/ARCHITECTURE.md)与[双库指南](docs/DUAL_DATABASES.md)。

健康采样的跨窗步数分配、累计计数日界与告警容差约定见[健康数据接口](智慧养老手表管理系统/backend/docs/health-data.md#跨窗口步数与夜间累计计数)。

## 源码与构建

开发从 `preview` 开始，复现发布版使用上方固定源码包。

```sh
git clone --branch preview https://github.com/WatchTestCs/smart-elderly-watch-management.git
cd smart-elderly-watch-management
```

```text
智慧养老手表管理系统/
├── desktop/     Tauri / Rust、原生权限与 Windows 打包
├── backend/     Go API、数据库迁移、照护与分析作业
├── oldIot/      Vue 3 / Vite、界面与浏览器 QA
└── watch-sim/   手表协议模拟器
deploy/         容器、数据库与运维配置
scripts/        平台启动、接口生成与性能工具
tests/          UI 冒烟检查
docs/           架构、部署、版本与验证记录
evidence/       历史测试原始证据
```

Windows 完整构建需 PowerShell 7、Node.js（`^20.19.0 || >=22.12.0`）、Go 1.23+、Rust 1.85+、MSVC 与 Windows SDK。依赖版本以模块声明和锁文件为准。

```powershell
cd 智慧养老手表管理系统
./desktop/validate-version.ps1
./desktop/build.ps1
```

构建顺序：**Vue 资源 → Go 内嵌服务 → sidecar SHA-256 → Tauri → NSIS**。产物位于工程目录 `outputs/v2.1.0-preview`；`desktop/package-release.ps1` 用于生成免安装包、源码与来源清单。前后端联调见[工程说明](智慧养老手表管理系统/README.md)，逐文件职责见[文件索引](docs/FILES.md)。`oldIot` 与 `careold-go` 为历史命名。

## 验证与性能

| 要核对的内容 | 入口与范围 |
| --- | --- |
| 当前分支自动检查 | [Preview CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml?query=branch%3Apreview)：前端测试、构建及依赖审计；Go 1.23.2 / 1.27.2 测试、竞争检测、构建和静态检查，使用专用 MySQL 8.4 / PG 16 |
| 2.1.0 发布检查 | [版本说明](docs/releases/2.1.0.md)与随包 `release-manifest.json`，以对应提交和实际检查为准 |
| 底层逻辑与职责整理 | [2026-10-26 重构](docs/refactoring/2026-10-26-boundaries/README.md)：会话取消与关闭、共享权限规则、JSON 请求合同和旧版问题复现 |
| 睡眠统计后续迭代 | [2026-10-26 睡眠优化](docs/performance/2026-10-26-sleep/README.md)：统一排序与阶段累计，含乱序/窗口外/稀疏样本和全部控制结果 |
| 算法与读取调度 | [2026-10-26 迭代](docs/performance/2026-10-26-algorithms/README.md)：统计内存、限流分片、命令查找及列表最新查询；含局部基准与未验证范围 |
| 后端长负载与缓存 | [2026-10-22 报告](docs/performance/2026-10-22-backend-hybrid/README.md)，保留合并前源码、隔离环境、延迟与队列积压边界 |
| 历史失败与原始数据 | [耐久测试记录](docs/performance/2026-10-21-go123/README.md)与 [CPU / GC 实验](docs/performance/2026-10-20-go123/README.md)，不以新结果覆盖旧失败 |

历史两轮 20 分钟 CI 耐久测试失败；后续本地修复版完成独立复测，当前合并源码未重跑同一长负载。CPU 微基准、缓存命中和真实落库数据不可互换，具体数字留在对应报告中。

在 `智慧养老手表管理系统/backend` 运行最小开发检查：

```sh
go test -count=1 ./...
go vet ./...
```

数据库集成需配置专用 `TEST_MYSQL_DSN` 与 `TEST_DATABASE_URL`，再运行 `go test -race -count=1 -timeout 600s ./...`；未配置时，跳过的用例不算已验证。20 分钟耐久测试需[显式启用](.github/workflows/mysql-endurance.yml)。完整要求见[测试分层](docs/TESTING.md)，禁止对生产库运行自动测试。

真实手表、外部通知网关、生产迁移、远程 TLS 和长期生产可用性仍需独立验收。健康与位置依赖有效上报；缺少新记录不代表状态正常，通知送达不等于处置完成。项目未取得医疗器械认证，不能替代人工应急响应。

## 常见问题

| 现象 | 首先核对 |
| --- | --- |
| 软件能打开，但无法登录或保存 | MySQL 连接、业务库名、迁移状态，以及 `SESSION_BACKEND` / `DATA_ENCRYPTION_KEY` |
| PG 离线，业务仍能保存 | 符合双库设计；监控分析积压和磁盘容量 |
| 升级后看不到旧记录 | 是否完成离线导入及图片复制；保留原备份 |
| 手表无法访问桌面服务 | 默认回环端口不对外，需独立规划设备接入 |
| 免安装版不能启动 | 完整解压、两个 EXE 同目录、已有 WebView2，核对附件版本与 SHA-256 |

## 文档导航

| 目的 | 文档 |
| --- | --- |
| 安装、版本与升级 | [下载索引](docs/DOWNLOADS.md) · [2.1.0 说明](docs/releases/2.1.0.md) · [更新日志](CHANGELOG.md) |
| 数据库、迁移与备份 | [双库指南](docs/DUAL_DATABASES.md) · [备份恢复](docs/runbooks/backup-restore.md) |
| 设备与照护运维 | [照护部署](docs/CARE_SERVICE_DEPLOYMENT.md) · [支持与排查](docs/SUPPORT.md) |
| 开发与接口 | [工程](智慧养老手表管理系统/README.md) · [前端](智慧养老手表管理系统/oldIot/README.md) · [后端](智慧养老手表管理系统/backend/README.md) · [构建](docs/BUILD.md) |
| 架构与验证 | [架构总览](docs/architecture/overview.md) · [历史决定](docs/decisions/README.md) · [测试](docs/TESTING.md) · [文件索引](docs/FILES.md) |

通过 [Issues](https://github.com/WatchTestCs/smart-elderly-watch-management/issues) 反馈问题时，请提供版本或提交、操作系统与脱敏复现步骤。贡献请基于 `preview`，遵循[贡献指南](CONTRIBUTING.md)；漏洞按[安全策略](.github/SECURITY.md)私下报告，不公开人员数据、连接串或密钥。

[MIT License](LICENSE)。字体、地图和其他第三方组件遵循各自许可证。
