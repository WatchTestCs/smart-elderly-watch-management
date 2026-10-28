# 岁宁 2.1.0 · 源码工程

本工程由 **Tauri 2 / Rust 桌面壳、Go 业务服务、Vue 3 / Vite 界面和手表模拟器**组成。MySQL 是唯一业务主库，PostgreSQL 保存异步分析投影；数据库由部署方独立管理。

[产品与整体架构](../README.md) · [2.1.0 发布](../docs/releases/2.1.0.md) · [双库配置与迁移](../docs/DUAL_DATABASES.md) · [实现协议](../docs/ARCHITECTURE.md) · [性能与数据库验收](../docs/performance/2026-10-21-go123/README.md)

## 工程职责

| 目录 | 职责 | 主要入口 |
| --- | --- | --- |
| `desktop` | 原生窗口、单实例、受限初始化、Go 哈希校验与生命周期 | `src/main.rs`、`capabilities`、`build.ps1` |
| `backend` | 业务 API、设备接入、会话、照护与分析作业 | `cmd/server`、`internal/handlers`、`internal/care`、`internal/analytics` |
| `backend/migrations` | MySQL 业务与旧 PG 迁移、PG 分析 schema | 内嵌 SQL 与版本校验 |
| `frontend` | Vue 路由、管理与家属页面、字体、地图、QA | `src`、`vite.config.js`、`qa` |
| `watch-sim` | 手表接口与运行端口发现模拟 | 工具及测试文件 |

Rust 启动 Go，Vue 从 Go 获取内嵌生产资源并调用同源 API；首次初始化走受限原生 IPC。Go 所有业务读写都使用 MySQL，分析作业将已提交 outbox 事件写入 PG，失败时保留队列。`frontend` 是现行 Vue 工程；生产环境使用 Go + MySQL，PostgreSQL 仅承担可选分析任务。

## Windows 完整构建

准备 PowerShell 7、Node.js（`^20.19.0 || >=22.12.0`）、Go、Rust、C++ Build Tools 和 Windows SDK。Go 模块声明最低 Go 1.26.0，CI 检查 Go 1.26.10 与 1.27.2。Rust 包声明最低 1.85。版本以锁文件和模块声明为准。

```powershell
./desktop/validate-version.ps1
./desktop/build.ps1
# 可选本机自签，不是公开受信任证书
./desktop/build.ps1 -SelfSign
```

构建顺序：锁定依赖安装 → Vue 生产构建 → 复制 Go 内嵌资源 → Go 测试、静态检查、编译 → sidecar 哈希 → Tauri 编译 → NSIS Setup。脚本支持 `CARGO_TARGET_DIR`；安装包位于工程目录 `outputs/v2.1.0-preview/Suining-Setup-2.1.0-win-x64.exe`。版本校验覆盖前端、桌面、Cargo、Go 版本文件及锁文件中的产品版本。

默认不签名。构建后运行 `desktop/package-release.ps1` 生成免安装 ZIP、源码、说明与摘要，并检查安装与启动。Setup 缺失 WebView2 时联网补装；免安装版需已有 WebView2，并保留同目录的两个 EXE。

## 前后端开发

在一个终端进入 `backend`，配置开发库并启动服务：

```powershell
$env:MYSQL_DSN='suining:替换密码@tcp(127.0.0.1:13306)/suining'
$env:PG_ANALYTICS_URL='postgres://suining_analytics:替换密码@127.0.0.1:15432/suining_analytics?sslmode=disable'
$env:ADDR='127.0.0.1:8795'
go run ./cmd/server
```

另一个终端进入 `frontend`：

```powershell
npm ci
npm run dev
```

Vite 将 `/api/v1` 和 `/openapi.json` 代理到 `127.0.0.1:8795`。纯 API 模式可直接开发；完整桌面构建必须先嵌入前端。纯浏览器开发不能通过 Tauri 初始化管理员，使用原生桌面完成首次初始化，不应放开 HTTP 初始化鉴权。

以上无 TLS 连接只适用于受控本机环境，远程连接需验证证书。PG 可暂不配置，核心业务仍可使用，但分析事件会积压，需监控容量。桌面使用 `SUINING_MYSQL_DSN` / `SUINING_PG_ANALYTICS_URL`，配置修改后重启；旧 `SUINING_DATABASE_URL` 不再作为主库回退。

## 测试、数据与迁移

- [后端 README](backend/README.md)：Go 测试、专用 MySQL / PG 集成环境、离线导入与分析回填。
- [前端 README](frontend/README.md)：生产构建、Playwright QA、会话与响应式页面。
- [双库指南](../docs/DUAL_DATABASES.md)：Compose、权限、事件重试、旧 PG 到空 MySQL 的迁移。安装不会自动迁移数据库；图片另行复制，旧 Java/H2 不直接支持。
- [照护部署](../docs/CARE_SERVICE_DEPLOYMENT.md)：长期值守、通知网关、送达回执与人工响应。桌面关闭后本地 Go 随之退出，不能替代常驻值守服务。

MySQL 不可用时界面仍能打开，业务 API 提示不可用；恢复后检查会话，不重放失败业务写入。PG 不可用只延迟分析。默认 Windows 图片、日志与运行信息位于 `%APPDATA%\岁宁`，会话默认在数据库中管理（明确设置 `SESSION_BACKEND=memory` 才使用单实例内存会话），业务库和图片分别备份。

本地 2.0.6 已构建并完成版本、摘要与内置服务检查，未执行安装器或原生桌面窗口验收。历史 Linux 附件不能证明本工程已完成 Linux 原生构建。仅更新 `preview`；此次整合保留并适配远程健康采样，见 [整合记录](../docs/INTEGRATION_2.0.6.md)。先前 Windows 安装包未包含该整合；[v2.0.6](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.6) 已发布仅含源码、校验文件和说明的 Release，**并未公开新的 Windows 安装包**。
