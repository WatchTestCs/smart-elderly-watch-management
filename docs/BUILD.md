# Windows 构建 · 2.0.2

准备 Windows x64、C++ Build Tools（MSVC + Windows SDK）、Rust stable、Go 1.27+、Node.js 22.12+、PowerShell 7。推荐在 Developer PowerShell 中构建。Tauri 与前端工具版本由 package-lock.json 固定，Rust 依赖由 Cargo.lock 固定。

```powershell
cd 智慧养老手表管理系统
./desktop/build.ps1
# 可选：明确标注为本机自签，不是公开受信任证书
./desktop/build.ps1 -SelfSign
```

按顺序执行前端安装/构建、嵌入静态资源、Go 测试及编译、生成 sidecar 哈希、Tauri 原生编译、NSIS 打包。默认从干净源码构建发布，避免将旧生成资源带入安装包。

产物为 `outputs/v2.0.2-preview/Suining-Setup-2.0.2-win-x64.exe`，附校验和及可选签名记录。安装器内含 WebView2 离线安装器，因此首次安装运行环境无需再联网；数据库、远程地图等外部服务不属于离线业务能力。

数据库不需要为编译运行；设置专用 `TEST_DATABASE_URL` 后才会执行真实数据库集成测试，未设置时会明确跳过，不能称为集成验收通过。直接 Go 开发可使用纯 API 模式；桌面发布必须嵌入真实前端。

源码不再要求 Maven、JDK 或 Electron。Tauri 官方参考：[Windows 前置条件](https://v2.tauri.app/start/prerequisites/)、[Windows 安装包](https://v2.tauri.app/distribute/windows-installer/)、[sidecar](https://v2.tauri.app/develop/sidecar/)。
