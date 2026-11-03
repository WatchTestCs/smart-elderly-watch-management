# Windows 构建 · 2.1.0

准备 Windows x64、C++ Build Tools（MSVC + Windows SDK）、Rust stable、Go 1.26.0+（CI 与容器镜像统一使用 1.27.2）、Node.js 22.12+、PowerShell 7。推荐在 Developer PowerShell 中构建。Tauri 与前端工具版本由 package-lock.json 固定，Rust 依赖由 Cargo.lock 固定。

```powershell
cd 智慧养老手表管理系统
./desktop/build.ps1
./desktop/package-release.ps1
# 可选：明确标注为本机自签，不是公开受信任证书
./desktop/build.ps1 -SelfSign
```

按顺序执行前端安装/构建、嵌入静态资源、Go 测试及编译、生成 sidecar 哈希、Tauri 原生编译、NSIS 打包。默认从干净源码构建发布，避免将旧生成资源带入安装包。

产物为 `outputs/v2.1.0-preview/Suining-Setup-2.1.0-win-x64.exe`，附校验和及可选签名记录。当前源码使用 `downloadBootstrapper`、`silent: true`：不附带 WebView2 运行时或引导程序。安装时检测已有组件，缺失时才联网下载微软引导程序并静默补装；下载或补装失败时安装器报错，不把缺少组件的状态当成成功。WebView2 安装为 Windows 共享组件，卸载岁宁不会移除它。缺少组件且无法联网时不能完成安装。当前构建默认未签名；脚本支持 `CARGO_TARGET_DIR`。

数据库不需要为编译运行；设置专用 `TEST_MYSQL_DSN` 和 `TEST_DATABASE_URL` 后才会执行真实数据库集成测试，未设置时会明确跳过，不能称为集成验收通过。直接 Go 开发可使用纯 API 模式；桌面发布必须嵌入真实前端。

Tauri 官方参考：[Windows 前置条件](https://v2.tauri.app/start/prerequisites/)、[Windows 安装包](https://v2.tauri.app/distribute/windows-installer/)、[sidecar](https://v2.tauri.app/develop/sidecar/)。

2.1.0 的 `package-release.ps1` 在隔离临时目录检查免安装和安装后启动、内嵌页面、父进程终止清理及卸载保留资料；附件保存在 `outputs/v2.1.0-preview/release`。使用有 WebView2 的 Windows 构建环境。该检查不连接生产数据库，也不代表完整可视验收。

## 容器构建入口

Dockerfile 位于 `deploy/Dockerfile`，构建上下文仍是仓库根目录，`.dockerignore` 保留在根目录。从仓库根目录执行：

```sh
docker build -f deploy/Dockerfile -t suining:preview .
```

`deploy/platform.compose.yml` 已使用同一 Dockerfile 路径；完整平台配置与密钥准备见项目 README。

## 签名

默认不签名。`-SelfSign` 使用本机证书，不自动安装根证书，也不消除 SmartScreen。
正式证书通过 `SUINING_SIGNING_THUMBPRINT` 和 `SUINING_TIMESTAMP_SERVER` 配置；`sign.ps1` 校验证书用途、私钥、有效期和时间戳。私钥、PFX 与密码不得进入仓库或 Release。

顺序：签 sidecar → 计算内嵌哈希 → 签主程序 → bundler `--no-binary-patching` → 签 Setup → 生成校验和。
