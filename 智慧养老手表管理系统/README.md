# 岁宁 · 智慧养老桌面软件

项目已迁移为 Vue 3 + Vite 8、Spring Boot 3.5 + 内置 H2 数据库，并封装为 Windows 10/11 x64 软件。安装包自带浏览器引擎和 Java，无需另装 Node、Java 或 MySQL。

完整使用说明、当前架构和构建流程见 [项目说明](智慧养老手表管理系统/README.md)。首次打开创建管理员账号，升级和卸载保留用户数据。

本地发布产物在 `智慧养老手表管理系统/outputs/desktop-release-20261015/release/`：Setup.exe、免安装 ZIP、整理后的源码 ZIP、公开自签证书及 SHA256 清单。此轮未推送 GitHub。

应用与安装/卸载程序已自签；自签不等于公开受信任的商业代码签名，不自动修改系统根信任。

验收日志与真实界面截图保存在 `智慧养老手表管理系统/outputs/desktop-release-20261015/`。旧文档、SQL、原始包与历史快照保留用于追溯；当前构建不依赖旧 Webpack/Babel/node-sass 或外部数据库启动器。
