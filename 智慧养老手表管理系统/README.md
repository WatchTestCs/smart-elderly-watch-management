# 岁宁 2.0.2 · 源码工程

本目录包含当前 Windows 桌面主线的四部分：Tauri / Rust 桌面壳、Go 业务服务、Vue 3 / Vite 界面和手表接口模拟器。业务数据使用独立 PostgreSQL，不由安装器创建或托管。

[项目介绍与下载](../README.md) · [架构](../architecture.md) · [构建](../docs/BUILD.md) · [使用与连接配置](../docs/SETUP.md)

## 工程分工

| 目录 | 内容 | 开发入口 |
| --- | --- | --- |
| `desktop` | Tauri、窗口、原生命令、sidecar 校验与 Windows 构建 | `src/main.rs`、`tauri.conf.json`、`build.ps1` |
| `backend` | Go HTTP、业务、会话、PostgreSQL、SQL 迁移 | `cmd/server`、`internal`、`migrations` |
| `oldIot` | Vue 页面、组件、字体、地图与前端 QA | `src`、`vite.config.js`、`qa` |
| `watch-sim` | 手表接口模拟与运行端口发现 | 模拟工具与测试文件 |

`oldIot` 和模块名 `careold-go` 为历史命名；当前不需要 Maven、JDK、Electron 或旧 Webpack/node-sass 工具链。

## 构建 Windows 安装包

准备 Windows x64、C++ Build Tools / Windows SDK、Rust stable、Go、Node.js 与 PowerShell 7。当前 CI 使用 Go 1.27.2；Go 模块最低声明为 1.24.0。前端 Node 支持范围以 `oldIot/package.json` 为准，构建指南建议 Node 22.12+。

在本目录执行：

```powershell
./desktop/build.ps1
# 可选本机自签名；不是公开受信任签名
./desktop/build.ps1 -SelfSign
```

脚本安装锁定依赖、构建 Vue、嵌入 Go 静态资源、执行 Go 测试与编译、生成 sidecar 哈希，再编译 Tauri 和生成 NSIS Setup。产物位于 `outputs/v2.0.2-preview`。缺少 WebView2 时，Setup 联网下载微软引导程序并静默补装共享运行时。

`desktop/version.json` 为产品版本入口；`validate-version.ps1` 核对各模块产品版本。源码 ZIP 和免安装 ZIP 是发布整理步骤，已有附件见 Release，不把这些步骤误称为 build.ps1 的自动输出。详细工具和环境说明见 [BUILD](../docs/BUILD.md)。

## 前后端分别开发

前端：进入 `oldIot`，执行 `npm ci`、`npm run dev`。开发代理将 `/restful` 转发到本机 8795 端口。

后端：进入 `backend`，先配置独立开发数据库；直接启动 Go 时使用 `DATABASE_URL`，并设置 `ADDR=127.0.0.1:8795` 与前端代理对应，再运行 `go run ./cmd/server`。不要使用生产库做回归测试。

纯浏览器开发不具备 Tauri 原生管理员初始化令牌；首次初始化需要原生桌面链路，不能通过放开 HTTP 初始化权限来绕过。生产桌面会加载 Go 内嵌的真实前端，不依赖 Vite 开发服务器。

## 连接与本地数据

桌面读取 `SUINING_DATABASE_URL`，传给 Go；连接串由部署方提供，不在源码中保存生产账号。修改配置后重启软件。自动重连只针对已经配置的数据库，不会自行寻找服务器或猜测凭据。

未连接数据库仍可打开界面。已登录后断线保留页面与输入，数据库相关操作就地提醒；退出登录、字号与窗口操作不依赖数据库。恢复后重新检查会话，不重放失败操作。

Windows 默认图片、日志及运行信息目录为 `%APPDATA%\岁宁`；业务数据库独立保存，会话位于 Go 内存。旧 H2 文件不会自动转换成 PostgreSQL，保留旧资料不代表已经完成迁移。首次空库创建管理员，没有默认通用账号。

## 测试与发布范围

[测试指南](../docs/TESTING.md) 区分 Go/真实 PostgreSQL、模拟 API 浏览器检查和原生桌面验收。未配置 `TEST_DATABASE_URL` 时数据库测试会跳过，不能称为集成通过。UI 模拟器与手表接口模拟不等于真实硬件联调。

Release 中 Linux 软件与数据库工具是用户提供的附件，不代表本目录的 Tauri 源码已在 Linux 完成编译和运行验证。见 [Linux 说明](../docs/LINUX.md)。

只在 `preview` 提交，本次文档更新不更改二进制、版本号、既有标签或仓库全局 About。
