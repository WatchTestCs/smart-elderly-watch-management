<p align="center"><img src="智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-192.webp" width="88" height="88" alt="岁宁标志"></p>

# 岁宁 · 智慧养老管理

**岁月有伴，日常安宁。**

岁宁面向照护机构与家属，将长者档案、手表设备、健康与活动记录、位置告警和服务公告放在同一工作空间。机构可以集中维护照护信息，家属可以查看关联家人的日常记录。

当前主线为 **Tauri 2 / Rust + Go + Vue 3 / Vite + 独立 PostgreSQL**。Windows 以本地桌面软件交付，数据库由部署方独立管理：没有数据库也能打开界面，相关业务功能就地提示不可用，网络与数据库恢复后自动重连。

[下载 2.0.2](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.2-preview.1) · [快速开始](#开始使用) · [架构](architecture.md) · [开发构建](docs/BUILD.md) · [Linux 说明](docs/LINUX.md) · [验收与限制](docs/DATABASE_AVAILABILITY.md)

> 当前源码与文档维护在 **preview**。历史版本的 Java、Electron 与 H2 说明不能作为本版构建依据；发布源码包对应 Release 固定提交，可能与持续更新的 preview 文档不同。

## 能做什么

| 场景 | 当前功能 |
| --- | --- |
| 长者与照护档案 | 长者资料、监护人信息、监测阈值和设备关联 |
| 设备管理 | 设备登记、密钥与绑定码、在线状态、上报记录和异常跟进 |
| 健康与日常记录 | 心率、单项血压、睡眠、步数及生活信息查询 |
| 位置与安全 | 最近上报位置、求助记录与告警处置 |
| 家属协作 | 邀请码注册、关联家人、家属账号与资料维护 |
| 公告与互动 | 服务公告、图文内容和评论管理 |
| 本地使用体验 | 大字切换、键盘操作、全屏、安装版与免安装版 |

管理员和家属使用不同的工作区与权限。健康、位置和在线状态取决于设备最近一次有效上报；没有新记录不代表状态正常。地图使用 Leaflet 与 OpenStreetMap 瓦片，地图显示需要相应网络服务可用。

## 下载与选择

当前软件版本为 **2.0.2**，发布标签为 `v2.0.2-preview.1`。

| 下载 | 适用方式 | 说明 |
| --- | --- | --- |
| [Windows x64 安装包](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Setup-2.0.2-win-x64.exe) | 常规安装，约 28.8 MiB | 检测已有 WebView2，仅缺失时联网静默补装 |
| [Windows x64 免安装包](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Portable-2.0.2-win-x64.zip) | 完整解压，约 30.7 MiB | 运行 `suining.exe`，保留同目录 `suining-server.exe`；需系统已有 WebView2 |
| [2.0.2 源码 ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Source-2.0.2-preview.1.zip) | 复核、开发与构建 | 精确对应发布提交 `46791c2675744acd0afd4e50992522b7c3c77b81` |
| [Linux amd64 软件包](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Linux-amd64.tar.gz) | 本地 Go 服务配合浏览器 | 用户提供的附件，不是当前 Windows 源码的 Linux 重编译 |
| [Linux 数据库管理工具](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/suining-db-manager_1.0.0_amd64.deb) | PostgreSQL 管理，组件版本 1.0.0 | 不包含 PostgreSQL 数据库服务，依赖 GTK 3 / WebKitGTK 4.1 |

Release 同时提供 SHA-256 校验清单、来源记录、中文说明及 Windows 源码依赖清单。Linux 附件与本仓库当前后端的源码、迁移和安全修复一致性尚未确认，未执行 Linux 安装及运行验收；请先阅读 [Linux 交付边界](docs/LINUX.md)。

## 开始使用

1. **选择 Windows 安装版或免安装版。** 最终用户无需安装 Go、Rust、Java 或 Node.js。安装版在缺少 WebView2 时需要联网；WebView2 是 Windows 共享界面组件，不会被放入岁宁目录，也不会随岁宁卸载。
2. **直接打开软件。** 没有数据库时也会显示登录界面；大字、密码显隐和窗口操作仍可使用，不会被全屏连接页挡住。
3. **由部署方准备 PostgreSQL。** 配置数据库地址和专用账号，桌面通过 `SUINING_DATABASE_URL` 读取连接串；配置修改后重启软件。已有环境会在网络或数据库恢复后自动重连，不需要每次手动连接。
4. **首次创建管理员。** 空库完成迁移后进入初始化流程。**没有默认手机号或通用密码。** 家属账号通过管理员提供的邀请码注册。
5. **开始接入与维护记录。** 管理员登记设备和照护对象，再根据实际设备接入方式上报数据；默认本机回环服务不是可直接供互联网手表访问的公共网关。

连接串含有敏感凭据，不要提交到 Git 或发在公开讨论中。数据库可以在独立主机上，也可以由本机独立服务提供；“与软件分离”不等于必须部署在同一台电脑。具体设置见 [SETUP](docs/SETUP.md)。

### 数据库暂不可用时

| 状态 | 软件行为 |
| --- | --- |
| 启动时未连接 | 正常显示界面，数据库相关操作提示暂不可用 |
| 已登录后断线 | 保留当前工作区、页面和输入，退出登录仍可用 |
| 连接恢复 | 自动重新检查数据库与会话；会话失效仍需重新登录 |
| 保存等操作失败 | 不自动重放，保留本地输入，由用户确认后重试 |

软件不会把“未连接”显示成“业务正常”，也不会生成假记录或维护离线业务写入队列。自动连接以事先配置的目标为准，不能仅凭联网推断服务器地址或凭据。详见 [数据库可用性设计](docs/DATABASE_AVAILABILITY.md)。

## 界面与阅读体验

![数据库未连接时的真实登录界面](docs/screenshots/tauri-2.0.2-offline.png)

![管理员照护工作区](docs/screenshots/tauri-2.0.2-main.png)

保留选定的山水标志、**Source Serif 4** 与**思源宋体（Source Han Serif）**，采用克制的绿、米白与清晰的层次。支持大字和键盘操作；桌面默认最大化，`F11` 切换全屏，`Escape` 退出全屏。工作区允许滚动，数据错误与空记录分别呈现。

截图来自实际 Tauri 程序，业务内容使用临时测试库，不代表生产记录或所有平台的显示效果。

## 当前技术栈

| 层次 | 技术 | 职责 |
| --- | --- | --- |
| 桌面 | Tauri 2、Rust、Windows WebView2 | 原生窗口、单实例、受限初始化命令与子进程生命周期 |
| 界面 | Vue 3、Vue Router 4、Element Plus、Vite 8 | 管理端与家属端页面、表单、路由和生产资源 |
| 请求与可视化 | Axios、ECharts 6、Leaflet | API 请求、健康记录图表与位置展示 |
| 本地服务 | Go、标准库 HTTP、pgx | 业务接口、设备接入、会话、文件处理和数据库连接 |
| 数据存储 | 独立 PostgreSQL、事务与内嵌 SQL 迁移 | 业务记录、关系约束与版本迁移 |
| Windows 交付 | NSIS、锁文件、SHA-256、可选 Authenticode | 安装与免安装交付、依赖和完整性记录 |

当前不是 Spring Boot / Vue 2 项目，也不再随包携带 Electron、Java 或 H2。保留 `oldIot`、`careold-go` 等历史目录或模块名称，不表示仍在使用旧运行时。

## 源码结构与开发

```text
智慧养老手表管理系统/
├── desktop/      Tauri / Rust、能力配置、版本与 Windows 构建
├── backend/      Go HTTP 服务、SQL 迁移与业务回归
├── oldIot/       Vue 3 / Vite 界面、本地字体与浏览器验收
└── watch-sim/    手表接口模拟工具

docs/            使用、架构、发布、验收、截图与逐文件介绍
```

Windows 源码构建需要 Node.js、Go、Rust、C++ Build Tools / Windows SDK 和 PowerShell 7；精确要求与锁定方式见 [BUILD](docs/BUILD.md)。完整交付从项目目录执行：

```powershell
cd 智慧养老手表管理系统
./desktop/build.ps1
# 本机自签名构建，不能替代公开受信任证书
./desktop/build.ps1 -SelfSign
```

Go 编译嵌入 Vue 生产资源与 SQL 迁移，不要求最终用户自行运行前端开发服务。开发接口代理和测试数据库的用法见 [工程说明](智慧养老手表管理系统/README.md) 与 [测试分层](docs/TESTING.md)。

## 数据、安全与验证边界

- 业务数据库独立备份；Windows 本地图片、日志及运行发现文件默认位于 `%APPDATA%\岁宁`。会话在 Go 进程内存中，程序重启后需要重新建立会话。
- 桌面初始化令牌不交给网页；普通业务写入使用会话、角色和 CSRF。设备接口另有设备鉴权。详细信任边界见 [实现架构](docs/ARCHITECTURE.md)。
- 旧版 H2 文件不会自动转换成 PostgreSQL。升级前备份旧安装包、用户目录与数据库；卸载保留资料不表示数据已完成迁移。
- 当前 Windows 包使用本机自签名，不具备公开 CA 信任或 SmartScreen 信誉。不会自动安装根证书。
- 已记录的 Windows 验收包括真实数据库断开/恢复、初始化、登录、退出、安装与卸载。浏览器模拟 API 检查不等于后端验收，本次文档更新没有重复执行这些测试。
- 未验证所有 Windows 版本、干净系统缺失 WebView2 时的实际补装、真实手表与短信、生产数据迁移及 Linux GUI 运行；没有性能基准或“零漏洞”保证。

## 文档导航

| 阅读目的 | 入口 |
| --- | --- |
| 安装、账号与连接配置 | [使用指南](docs/SETUP.md) |
| 整体设计与开发定位 | [架构总览](architecture.md) · [实现与协议](docs/ARCHITECTURE.md) |
| 构建与测试 | [Windows 构建](docs/BUILD.md) · [测试说明](docs/TESTING.md) |
| 独立数据库与自动恢复 | [数据库可用性](docs/DATABASE_AVAILABILITY.md) |
| Linux 附件 | [Linux 使用说明](docs/LINUX.md) |
| 备份、回退和签名 | [运行手册](docs/runbooks/backup-restore.md) · [回退](docs/runbooks/rollback.md) · [签名](docs/runbooks/signing.md) |
| 历史决定与文件职责 | [ADR](docs/decisions/README.md) · [逐文件索引](docs/FILES.md) |
| 仓库 About 文案 | [简介与标签建议](docs/ABOUT.md) |

只在 `preview` 分支维护和提交本轮工作；发布操作按每次明确授权执行。最新 2.0.2 的补充附件与持续更新的文档分开追踪，不因文档更新重打二进制或移动发布标签。

[MIT License](LICENSE)。字体、系统运行时与第三方组件遵循各自许可。反馈问题时请附软件版本、操作系统、安装方式及复现步骤，不要公开账号密码、连接串或真实照护人员数据。
