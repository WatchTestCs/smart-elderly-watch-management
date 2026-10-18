# 岁宁 · 2.0.2 预览版

岁月有伴，日常安宁。面向照护机构与家属的 Windows 桌面软件，管理长者档案、设备、健康记录、位置、求助和服务公告。

**当前技术栈：Tauri 2 + Go + Vue 3，数据库独立部署为 PostgreSQL。** 软件无需数据库也能直接显示界面；只有登录、查询、保存等依赖数据库的功能提示暂不可用。网络恢复后自动尝试连接已配置的数据库，不使用全屏连接页阻挡软件。不是内置数据库软件，离线状态不提供虚构的业务记录。

[下载 2.0.2 安装包](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.2-preview.1) · [使用说明](docs/SETUP.md) · [构建指南](docs/BUILD.md) · [本版变化](docs/releases/2.0.2.md) · [验收与限制](docs/VALIDATION.md)

> 按用户要求原位更新 2.0.2：Windows 改为轻量安装，补充 Linux 软件包及数据库管理工具，保留版本号。

## 平台下载

| 文件 | 用途 |
| --- | --- |
| [Windows x64 安装包](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Setup-2.0.2-win-x64.exe) | 约 28.8 MiB，缺失 WebView2 时联网静默补装 |
| [Linux amd64 软件包](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Linux-amd64.tar.gz) | 用户提供的 Go 服务与浏览器启动脚本，不是 Windows Tauri 包的 Linux 重编译 |
| [Linux 数据库管理工具 1.0.0](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/suining-db-manager_1.0.0_amd64.deb) | PostgreSQL 管理工具，不包含 PostgreSQL 数据库服务 |

Linux 安装与验收边界见 [Linux 使用说明](docs/LINUX.md)。软件和数据库仍独立部署，数据库地址需预先配置。

## 安装与首次使用

下载 `Suining-Setup-2.0.2-win-x64.exe`。本次替换的 2.0.2 Windows 安装器使用轻量安装：检测并复用系统 WebView2，仅在缺失时联网下载并静默补装。不要求用户另装 Java、Node.js、Rust 或 Go。WebView2 为 Windows 共享组件，不放入软件目录。

打开后若数据库未运行，仍显示登录界面；已登录时保留工作区、页面和输入。数据库相关功能显示就地提醒，后台自动重连；字体切换、窗口操作与已登录会话的退出不依赖数据库。部署方准备独立 PostgreSQL 后设置 `SUINING_DATABASE_URL`，重新打开软件；数据库连通后自动检查会话及首次初始化状态，失败操作不自动重放。**没有默认手机号或密码**，空库首次使用由您创建管理员。

本版切换了桌面与后端架构。**旧版 H2 文件不会自动转换成 PostgreSQL**；升级前备份旧数据，已有业务数据需单独安排迁移。保留 `%APPDATA%\岁宁` 中原有文件不等于已经迁移了旧数据库。

## 界面

![数据库未连接时仍显示登录界面](docs/screenshots/tauri-2.0.2-offline.png)

![岁宁照护界面](docs/screenshots/tauri-2.0.2-main.png)

沿用现有山水标志、Source Serif 4 与思源宋体；桌面默认最大化，F11 切换全屏、Escape 退出全屏。数据库离线时只展示明确状态；需要数据库的写入返回失败，不自动重放。

## 本版改动

- 使用 Tauri 原生桌面壳，移除 Electron、随包 Java 和旧构建脚本。
- Go 服务随安装包启动，迁移 SQL 与实际前端资源编入后端，支持任意工作目录启动。
- 保留独立 PostgreSQL 部署；本机启动、退出和连接提示不等待数据库。
- 修复图片预解码检查、家属绑定竞争、退出权限、坐标和 SOS、列表契约及编辑误解绑。
- 桌面初始化通过受限原生命令完成，令牌不暴露给网页；本地服务仅接受受控页面的初始化请求。

## 开发与交付

源码只在 `preview` 分支维护。发布标签 `v2.0.2-preview.1` 指向对应 `preview` 提交，后续发布按最小版本递增。

- [架构](docs/ARCHITECTURE.md) / [架构决定](docs/decisions/README.md)
- [测试](docs/TESTING.md) / [文件索引](docs/FILES.md)
- [发布与签名](docs/RELEASE.md) / [备份与恢复](docs/runbooks/backup-restore.md)

目标系统为 Windows 10/11 x64；本轮以本机 Windows 环境验收，不能把它表述为所有 Windows 版本或真实手表均已验证。自签名不等于公开受信任证书，不会安装根证书或消除 SmartScreen 提示。

[MIT 许可](LICENSE)。第三方运行时、字体及组件遵守各自许可；源码树不包含用户数据库、私钥、安装包或编译缓存。
