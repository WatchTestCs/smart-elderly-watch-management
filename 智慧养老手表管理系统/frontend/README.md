# 岁宁前端 · 2.1.0

岁宁前端基于 Vue 3、Vue Router 4、Element Plus 2 与 Vite 8 构建，使用 Axios 调用 Go 后端接口，通过 ECharts 6 渲染健康趋势图表，并集成 Leaflet 与 OpenStreetMap 展示定位与轨迹。在 Windows 桌面端中，Tauri 2（WebView2）直接加载编译内嵌在 Go 服务端中的生产静态资源。

[整体技术栈与架构](../../README.md) · [源码工程总览](../README.md) · [后端接口与数据](../backend/README.md) · [历史改进记录](../../CHANGELOG.md)

## 模块职责

- 管理端与家属端共享基础 UI 组件、服务端会话及 CSRF 校验机制，家属账号登录后仅展示已绑定长者的数据。
- 侧边导航按当前路由自动展开对应分组；长者档案页在窄屏下自动切换为资料卡片视图，运营总览在宽屏下采用栅格布局，列表与表单页面均内置加载状态与失败重试操作。
- 照护协作模块提供工单与履历分页、接单、处置登记、双人独立复核及队列诊断界面，并对护理员与审计员账号实行按角色受限访问。
- `src/api/auth.js` 负责跨页面轮询服务就绪状态。当数据库断开或尚未就绪时，界面会保留当前路由与用户输入内容，提示手动重试而不自动重放写请求。
- 首次部署的管理员初始化界面通过 `src/desktop.js` 调用受限的 Tauri 原生命令，前端代码不保存桌面初始化令牌或数据库凭据。

前端所有数据均通过 Go `/api/v1/` 接口读写，不直接连接 MySQL 或 PostgreSQL。

## 本地开发与构建

请使用 Node.js `^20.19.0 || >=22.12.0`，并通过 `package-lock.json` 安装固定版本依赖：

```powershell
npm ci
npm run dev
# 构建生产静态产物至 dist/
npm run build
```

本地开发时 Vite 仅绑定本机回环地址，并将 `/api/v1` 代理至 `http://127.0.0.1:8795`。启动 Go 后端时设置 `ADDR=127.0.0.1:8795` 、`MYSQL_DSN` 及 `DATA_ENCRYPTION_KEY_FILE` 后即可联调业务页面（首次初始化需使用桌面引导或仓库初始化脚本）。

生产构建以 Chrome 120 为目标环境，默认关闭 source map 并拆分 CSS。执行桌面打包脚本 `desktop/build.ps1` 时，会自动将 `dist/` 下的产物复制到 Go 内嵌资源目录。

## 浏览器端测试与静态资源

```powershell
npm --prefix qa ci
npm --prefix qa test
```

`qa/` 目录下的端到端测试基于 Playwright 编写，默认拦截模拟 API 以验证页面交互、空状态与窄屏布局；如需针对真实后端运行验证，请先准备专用的空 MySQL 测试库。

品牌图标、字体与配图位于 `public/static/`。在线地图瓦片依赖外部网络连接；对于设备尚未上报的健康指标，界面会直接显示缺失状态，不会用 `0` 填充。

## API v1 路径与会话切换说明

所有浏览器、桌面端与手表端请求统一使用 `/api/v1/` 前缀，后端已停止接收无版本号的 `/api/` 请求。服务端会话 Cookie 名称已更新为 `SUINING_SESSION`，升级后旧会话需重新登录。
