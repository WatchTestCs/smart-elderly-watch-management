# 测试分层 · 2.0.6

- Go 回归：`go test ./...`、`go vet ./...`；专用双库配置 `TEST_MYSQL_DSN` 与 `TEST_DATABASE_URL` 后执行 `go test -race -count=1 -timeout 240s ./...`。事务竞争、外键、SOS 和真实接口必须使用实际数据库。
- Vue：`npm ci && npm run build`；frontend/qa 使用 Playwright 的模拟接口验收布局和交互，不能充当后端验收。保留字体及窄屏检查。
- Rust：准备 sidecar 与哈希后 `cargo test --locked`，Tauri 原生命令、窗口 origin 和进程生命周期还需运行验收，不能用编译替代。
- Windows 安装：运行本版 Setup，确认真实安装路径、软件无数据库启动、恢复连接、首次初始化、登录、全屏与退出。当前轻量安装器缺失 WebView2 时联网静默补装；应分别验证已有组件和干净系统缺失组件两条路径。本机已有运行环境时，不能据此声称验证了缺失时的实际下载安装。

禁止对生产数据库运行自动测试。MySQL 用例创建并清理随机测试数据库，PG 用例使用随机 schema 或独立分析库；账号需要对应权限。失败业务写入不得自动重放，分析事件允许幂等重试；退出必须结束 sidecar。

2.0.6 健康采样整合保留版本冲突、整批回滚、家属隔离、游标提交顺序、绑定版本和统计校验。旧库导入额外覆盖 PG 0006／0007／0008 的预检查、健康记录与修订时钟保留、拒绝非空目标及失败回滚。执行证据见 [整合记录](INTEGRATION_2.0.6.md)，历史状态仍保留在 [VALIDATION](VALIDATION.md)。
