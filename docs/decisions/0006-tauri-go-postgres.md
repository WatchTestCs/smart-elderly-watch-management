# ADR-0006：Tauri / Go / 独立 PostgreSQL

- Status: Accepted
- Date: 2026-10-18
- Supersedes: ADR-0001, ADR-0002, ADR-0003

用户要求使用 Tauri 发布 2.0.2，并明确数据库与软件分离是策略。桌面改为 Rust + WebView2，随包启动已修复的 Go 服务，业务数据库采用独立 PostgreSQL。

软件界面与退出不依赖数据库；真实业务功能需数据库就绪。迁移 SQL 和前端产物内嵌 Go，不意味着数据库也内嵌。Rust 负责 sidecar 校验、生命周期和受限初始化桥；Go 负责业务、会话、CSRF 与事务，网页不获得桌面令牌。

2.0.2 最初发布的 NSIS Setup 包含 WebView2 离线安装器。根据后续用户要求，当前源码改用下载引导程序：复用已有组件，仅缺失时联网静默补装 Windows 共享 WebView2，不再随包携带运行时。旧 H2 文件不会自动转换，升级需先备份并单独安排迁移。字体、UI 与 preview 发布约束继续有效。取消旧实现不删除 Git 历史或旧 Release。
