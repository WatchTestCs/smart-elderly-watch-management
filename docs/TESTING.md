# 测试分层 · 2.0.2

- Go 回归：`go test ./...`、`go vet ./...`；专用 PostgreSQL 配置 `TEST_DATABASE_URL` 后执行 `go test -race -count=1 -timeout 120s ./...`。事务竞争、外键、SOS 和真实接口必须使用实际数据库。
- Vue：`npm ci && npm run build`；oldIot/qa 使用 Playwright 的模拟接口验收布局和交互，不能充当后端验收。保留字体及窄屏检查。
- Rust：准备 sidecar 与哈希后 `cargo test --locked`，Tauri 原生命令、窗口 origin 和进程生命周期还需运行验收，不能用编译替代。
- Windows 安装：运行本版 Setup，确认真实安装路径、软件无数据库启动、恢复连接、首次初始化、登录、全屏与退出。安装器包含离线 WebView2 但本机已有运行环境时不能据此声称验证了全新 Windows 的运行时安装。

禁止对生产数据库运行自动测试。每个集成用例只创建并清理自己的随机 schema。失败写入不得自动重放；退出必须结束 sidecar。执行状态与实际限制记录在 [VALIDATION](VALIDATION.md)。
