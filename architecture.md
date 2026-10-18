# 岁宁开发架构入口 · 2.0.2

当前实现为 Tauri 2 / WebView2 + Go + Vue 3，业务数据库独立 PostgreSQL。详见 [当前结构](docs/ARCHITECTURE.md) 和 [ADR-0006](docs/decisions/0006-tauri-go-postgres.md)。旧 Java/H2/Electron 文档仅作历史，不是当前构建依据。

维护时先核对 `preview` 与版本，阅读相关契约，最小修改，执行真实验证，再更新证据。未执行的检查不能写成通过；模拟 API 的浏览器验收不能替代真实数据库及原生安装验收。

安全不变量：桌面令牌不进入网页；业务操作不越权；离线不表示正常；失败写入不重放；迁移不静默删除数据；签名状态如实披露。数据库故障不得阻止软件界面及正常退出。

[构建](docs/BUILD.md) · [测试](docs/TESTING.md) · [发布](docs/RELEASE.md) · [备份](docs/runbooks/backup-restore.md) · [文件用途](docs/FILES.md)。仅在 `preview` 提交，发布授权按每次请求确认。
