# Windows 构建 · 2.0.2

准备 Windows x64、C++ Build Tools（MSVC + Windows SDK）、Rust stable、Go 1.27+、Node.js 22.12+、PowerShell 7。推荐在 Developer PowerShell 中构建。Tauri 与前端工具版本由 package-lock.json 固定，Rust 依赖由 Cargo.lock 固定。

```powershell
cd 智慧养老手表管理系统
./desktop/build.ps1
# 可选：明确标注为本机自签，不是公开受信任证书
./desktop/build.ps1 -SelfSign
```

按顺序执行前端安装/构建、嵌入静态资源、Go 测试及编译、生成 sidecar 哈希、Tauri 原生编译、NSIS 打包。默认从干净源码构建发布，避免将旧生成资源带入安装包。

产物为 `outputs/v2.0.2-preview/Suining-Setup-2.0.2-win-x64.exe`，附校验和及可选签名记录。当前源码使用 `downloadBootstrapper`、`silent: true`：不附带 WebView2 运行时或引导程序。安装时检测已有组件，缺失时才联网下载微软引导程序并静默补装；下载或补装失败时安装器报错，不把缺少组件的状态当成成功。WebView2 安装为 Windows 共享组件，卸载岁宁不会移除它。缺少组件且无法联网时不能完成安装。已发布的 2.0.2 离线包不受本次源码调整影响。

数据库不需要为编译运行；设置专用 `TEST_DATABASE_URL` 后才会执行真实数据库集成测试，未设置时会明确跳过，不能称为集成验收通过。直接 Go 开发可使用纯 API 模式；桌面发布必须嵌入真实前端。

源码不再要求 Maven、JDK 或 Electron。Tauri 官方参考：[Windows 前置条件](https://v2.tauri.app/start/prerequisites/)、[Windows 安装包](https://v2.tauri.app/distribute/windows-installer/)、[sidecar](https://v2.tauri.app/develop/sidecar/)。

本次轻量打包本地验收：安装包约 28.82 MiB，已实际静默安装、启动已安装的 Tauri 软件、确认无数据库仍可打开、退出结束 Go 服务，卸载保留测试用户资料。生成的 NSIS 配置为 `downloadBootstrapper` 和 `/silent`，嵌入引导程序与离线安装器路径均为空；缺失时使用微软 HTTPS 下载地址，下载或执行失败会中止安装。本机已有 WebView2，未执行干净系统上的缺失补装测试，未卸载系统共享组件。本次按用户要求将轻量安装包替换到现有 2.0.2 Release。
