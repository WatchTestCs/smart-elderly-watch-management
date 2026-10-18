# 岁宁 · 2.0.2 预览版

岁月有伴，日常安宁。面向照护机构与家属的 Windows 桌面软件，管理长者档案、设备、健康记录、位置、求助和服务公告。

**当前技术栈：Tauri 2 + Go + Vue 3，数据库独立部署为 PostgreSQL。** 软件无需数据库也能打开；数据库准备好后自动继续。不是内置数据库软件，离线状态不提供虚构的业务记录。

[下载 2.0.2 安装包](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.2-preview.1) · [使用说明](docs/SETUP.md) · [构建指南](docs/BUILD.md) · [本版变化](docs/releases/2.0.2.md) · [验收与限制](docs/VALIDATION.md)

## 安装与首次使用

下载 `Suining-Setup-2.0.2-win-x64.exe`。安装器包含 WebView2 离线安装器，不要求用户另装 Java、Node.js、Rust 或 Go；这些工具仅构建源码时需要。

打开后若数据库未运行，软件显示连接状态并后台重试，不会直接退出。部署方准备独立 PostgreSQL 后设置 `SUINING_DATABASE_URL`，重新打开软件；数据库连通后自动进入初始化或登录页面。**没有默认手机号或密码**，空库首次使用由您创建管理员。

本版切换了桌面与后端架构。**旧版 H2 文件不会自动转换成 PostgreSQL**；升级前备份旧数据，已有业务数据需单独安排迁移。保留 `%APPDATA%\岁宁` 中原有文件不等于已经迁移了旧数据库。

## 界面

![数据库未启动也可打开软件](docs/screenshots/tauri-2.0.2-offline.png)

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
