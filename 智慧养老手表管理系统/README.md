# 岁宁 2.1.0 · 核心源码工程

本目录包含岁宁的 Tauri 2 / Rust 桌面壳、Go 业务后端、Vue 3 / Vite 前端界面以及 Node.js 手表协议模拟器。系统以独立部署的 MySQL 作为唯一业务主库，使用 PostgreSQL 异步存储分析投影。

[产品与整体架构](../README.md) · [2.1.0 发布说明](../docs/releases/2.1.0.md) · [双库配置与迁移](../docs/DUAL_DATABASES.md) · [架构与接口边界](../docs/ARCHITECTURE.md) · [测试与数据库验收](../docs/TESTING.md)

## 目录职责

| 目录 | 职责范围 | 核心入口 |
| --- | --- | --- |
| `desktop` | 原生窗口管理、单实例锁、受限初始化 IPC、Go sidecar 哈希校验与进程生命周期 | `src/main.rs`、`capabilities`、`build.ps1` |
| `backend` | 业务 API、设备鉴权接入、服务端会话、照护工单与异步分析作业 | `cmd/server`、`internal/handlers`、`internal/care`、`internal/analytics` |
| `backend/migrations` | MySQL 业务库迁移、旧版 PG 兼容迁移与 PG 分析库 schema | 内嵌 SQL 脚本与版本校验 |
| `frontend` | Vue 路由、管理端与家属端页面、本地静态字体与图标、Playwright 浏览器测试 | `src`、`vite.config.js`、`qa` |
| `watch-sim` | 手表 HTTP 上报模拟与本地运行端口自动发现 | `watch-sim.js`、`watch-sim.test.cjs` |

桌面启动时，Rust 主进程先校验并拉起 Go 子进程，Vue 界面从 Go 服务加载内嵌静态资源并调用同源 HTTP 接口；首次创建管理员则通过受限的 Tauri 原生 IPC 完成。所有实时业务读写均落在 MySQL，后台分析协程将已提交的 outbox 事件异步投影至 PostgreSQL，遇到 PG 离线时在队列中保留并自动重试。

## Windows 完整构建

构建前请安装 PowerShell 7、Node.js（`^20.19.0 || >=22.12.0`）、Go（`go.mod` 要求 `1.26.0+`）、Rust 1.85+、C++ Build Tools 和 Windows SDK。

```powershell
./desktop/validate-version.ps1
./desktop/build.ps1
# 如需在本地生成自签名测试包（非受信任 CA 证书）：
./desktop/build.ps1 -SelfSign
```

构建脚本会按顺序安装锁定依赖、执行 Vue 生产构建、将静态产物复制至 Go 内嵌目录、运行 Go 测试与编译、生成 sidecar SHA-256 摘要、编译 Tauri 桌面壳，最后打包 NSIS 安装程序。若需额外生成免安装 ZIP、源码归档与校验清单，可继续运行 `desktop/package-release.ps1`。当目标机器缺少 WebView2 运行时，Setup 安装包会自动联网下载安装。

## 前后端联调开发

打开第一个终端进入 `backend`，配置开发库连接并启动 Go 服务：

```powershell
$env:MYSQL_DSN='suining:替换密码@tcp(127.0.0.1:13306)/suining'
$env:PG_ANALYTICS_URL='postgres://suining_analytics:替换密码@127.0.0.1:15432/suining_analytics?sslmode=disable'
$env:ADDR='127.0.0.1:8795'
$env:DATA_ENCRYPTION_KEY_FILE='替换为已备份的持久化密钥文件路径'
go run ./cmd/server
```

打开第二个终端进入 `frontend` 启动 Vite 开发服务器：

```powershell
npm ci
npm run dev
```

Vite 会将 `/api/v1` 与 `/openapi.json` 请求代理至 `http://127.0.0.1:8795`。纯浏览器开发模式下无法调用 Tauri 原生 IPC（如桌面首次初始化接口），其余业务接口均可正常联调。

上述示例关闭了数据库连接 TLS，仅限本机回环开发使用；跨机器连接数据库请启用证书校验。开发阶段可以暂不配置 PostgreSQL，但需留意 MySQL 内的分析事件积压情况。桌面端调试如需启用明文 HTTP，可设置环境变量 `SUINING_ALLOW_HTTP=1`。

## 测试、数据与运维参考

- [后端开发说明](backend/README.md)：涵盖 Go 单元测试、MySQL / PG 真实数据库集成测试、旧库离线导入与分析回填命令。
- [前端开发说明](frontend/README.md)：涵盖生产打包、Playwright 浏览器测试、会话探测与响应式布局。
- [双库运维指南](../docs/DUAL_DATABASES.md)：说明容器编排、账号权限、outbox 重试机制，以及从旧版 PG 导入空 MySQL 的操作步骤。
- [照护服务部署](../docs/CARE_SERVICE_DEPLOYMENT.md)：说明生产环境常驻后台、外部通知网关、回执签名校验与人工值班流程。

当 MySQL 暂不可用时，桌面界面仍可正常打开并提示连接状态，待数据库恢复后自动重验会话。Windows 桌面端默认将图片与日志保存在 `%APPDATA%\岁宁`，可通过 `SUINING_USER_DATA` 指定其他数据目录。
