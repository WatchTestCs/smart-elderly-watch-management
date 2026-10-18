# 岁宁前端 · 2.0.2

Vue 3 + Element Plus + Vite 8，管理端和家属端共用界面基础、服务端会话与 CSRF 保护。

- 安装锁定依赖：`npm ci`
- 本地开发：`npm run dev`，接口代理到本机 8795 端口
- 生产构建：`npm run build`

Windows 软件的 Tauri 运行环境、独立 PostgreSQL 配置和打包流程见项目根目录 README 与 desktop/build.ps1。
本地字体、选定 Logo 和风景图保存在 public/static。测试记录与历史构建目录归档在 outputs，构建不依赖旧 Webpack/Babel/node-sass。

数据库离线不再转入阻断式连接页：登录界面与已登录工作区保留，数据库相关功能就地提醒，联网后自动重试已配置的连接。首次初始化通过受限 Tauri 原生命令。浏览器开发代理需要 Go 设置 `ADDR=127.0.0.1:8795`；纯浏览器模式不携带桌面初始化令牌。
