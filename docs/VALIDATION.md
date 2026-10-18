# 已发布 2.0.2 的历史本地验收记录

后续源码已改为非阻断数据库连接提醒，当前行为与验收见 [数据库可用性](DATABASE_AVAILABILITY.md)。以下记录保留已发布安装包的真实历史结果。

2026-10-18，Windows 本机完成 Tauri / Go / 独立 PostgreSQL 交付验收。以下为实际执行结果，历史 Java/Electron 报告不作为本版证据。

| 验收项 | 结果与范围 |
| --- | --- |
| 版本一致性 | desktop/version.json、Cargo、Tauri、前端及 backend/VERSION 均为 2.0.2 |
| 前端生产构建 | npm ci、Vite build 成功，沿用现有字体与视觉资产 |
| Windows Go | Go 1.27.2，go test -count=1 ./... 与 go vet ./... 成功，使用专用 PostgreSQL |
| 浏览器回归 | 生产前端，8 组共 1219 项检查通过；API 为模拟，不能代替真实 Go/PostgreSQL 验收 |
| 字体完整性 | 88 个字体文件与原 preview 逐字节一致 |
| Rust 发布构建 | Windows MSVC 原生 release 编译成功，NSIS 打包成功 |
| Go 竞争检测 | WSL/Linux Go 1.24.0 + 专用 PostgreSQL，go test -race -count=1 -timeout 120s ./... 全包通过 |
| Rust 测试与格式 | cargo test --locked 编译通过（0 个 Rust 单元用例），cargo fmt --check 通过；功能以真实安装后的验收为准 |
| Setup 安装 | 最终安装包实际静默安装，注册版本 2.0.2，主程序与 Go 服务存在 |
| 真实已安装应用 | 11 项原生检查全部通过，详见 verification.json |
| 数据库异常 | 无数据库可打开，离线 API 503，恢复后同进程继续，失败写入不重放 |
| 初始化安全 | 无桌面令牌被拒绝；原生命令初始化成功，随后真实登录成功 |
| 生命周期 | F11/Escape 正常；在线和离线关闭均结束 Go 子进程 |
| 页面错误 | 无未捕获页面异常；故意断库产生的两条 503 控制台记录符合预期 |
| 卸载 | 最终 Setup 对应卸载器移除程序与注册项，保留测试用户资料 |
| 签名 | 自签名，验证报告为不受信任根；未安装根证书，未使用公开 CA 或时间戳 |

实际截图：[离线启动](screenshots/tauri-2.0.2-offline.png)、[管理员首页](screenshots/tauri-2.0.2-main.png)。机器可读结果见 [verification.json](verification.json)，版本说明见 [2.0.2](releases/2.0.2.md)。

安装包包含 WebView2 离线安装器；宿主机已有 WebView2，不能据此声称全新无运行时 Windows 已通过。未验证所有 Windows 版本、真实手表、短信、生产迁移或公开证书信任，未进行性能测量。没有手动运行或查询远程 Actions；后续 CI 结果不得冒充本轮本地验收。

未验证旧 Electron/H2 安装的原位覆盖升级，没有 GitHub 构建来源证明。
