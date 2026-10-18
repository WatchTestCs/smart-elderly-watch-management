# 岁宁 2.0.2 源码工程

`desktop` 为 Tauri 2 / Rust 桌面壳，`backend` 为 Go / PostgreSQL 服务，`oldIot` 为 Vue 3 / Vite 界面，`watch-sim` 为手表模拟器。

Windows 构建：`./desktop/build.ps1`；本机自签构建可使用 `./desktop/build.ps1 -SelfSign`。输出在 `outputs/v2.0.2-preview`，默认产出含 WebView2 离线安装器的 NSIS Setup EXE。

构建需要 Windows C++ Build Tools / SDK、Rust stable、Go 1.27+、Node.js（范围见前端 package.json）与 PowerShell 7。完整步骤见 [BUILD](../docs/BUILD.md)。最终用户无需这些开发工具。

业务数据库独立部署。未配置或未启动数据库也能打开软件，连通后自动继续；离线时不能读取或写入真实业务记录。部署变量为 `SUINING_DATABASE_URL`，直接启动 Go 服务时使用 `DATABASE_URL`。

本机图片、日志与运行发现目录沿用 `%APPDATA%\岁宁`。旧 H2 文件不会自动迁移至 PostgreSQL；升级和迁移需先备份。没有默认管理员手机号或密码。
