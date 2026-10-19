# 岁宁前端 · 2.0.6

**Vue 3.5 + Vue Router 4 + Element Plus 2 + Vite 8**。Axios 调用 Go API，ECharts 6 展示健康记录，Leaflet / OpenStreetMap 展示位置；桌面通过 Tauri 2 / WebView2 加载 Go 内嵌的生产界面。

[整体技术栈与架构](../../README.md) · [工程开发与构建](../README.md) · [后端接口与数据](../backend/README.md) · [2.0.6 改进](../../docs/releases/2.0.6.md)

## 前端职责

- 管理与家属工作区共用基础组件、服务端会话与 CSRF；家属读取关联家人的记录。
- 当前路由所在导航组展开，术语与标题统一；长者档案在窄屏使用资料卡片，宽屏总览使用栅格，相关管理页提供加载与重试。
- 照护组件支持任务与履历分页、接单、处置、独立复核及诊断；护理员／审计员的岗位限制范围仍为照护功能。
- `auth.js` 统一探测服务状态并处理恢复；数据库未连接时保留登录界面或工作区和输入，不自动重放失败保存。
- 首位管理员初始化通过 `desktop.js` 调用受限原生命令，前端不持有桌面初始化令牌或数据库凭据。

Vue 不直接连接 MySQL 或 PostgreSQL。业务 API 由 Go 读写 MySQL；PG 只接收服务端异步分析投影，当前图表仍使用主库业务接口，不能把它们描述为直接查询 PG 报表。

## 开发与生产构建

Node.js 支持范围为 `^20.19.0 || >=22.12.0`，依赖以 `package-lock.json` 固定。

```powershell
npm ci
npm run dev
# 生成 dist
npm run build
```

Vite 仅绑定本机，`/restful` 代理到 `http://127.0.0.1:8795`。后端设置 `ADDR=127.0.0.1:8795`，配置 `MYSQL_DSN` 与可选 `PG_ANALYTICS_URL` 后启动。纯浏览器模式没有 Tauri 初始化能力，首次管理员初始化需原生桌面链路。

生产构建以 Chrome 120 为目标，关闭 source map，输出分割 CSS。桌面完整构建由 `desktop/build.ps1` 将 `dist` 复制到 Go 内嵌目录再编译；`vite preview` 只用于检查前端产物，不提供真实业务后端。

## 浏览器验收与资源

```powershell
npm --prefix qa ci
npm --prefix qa test
```

QA 使用 Playwright，需对应浏览器运行时；大部分页面验收拦截 API，不代表真实数据库、通知网关或原生窗口验收。真实后端脚本需专用空 MySQL、独立服务、测试令牌及 `CARE_E2E_TEST_ONLY=1`，仅允许 `qa_arch_` 前缀测试库；具体见 [双库复跑要求](../../docs/DUAL_DATABASES.md#复跑与-ci)。

Logo、字体和风景图在 `public/static`，当前不依赖旧 Webpack、Babel 或 node-sass。地图瓦片需要网络；未上报指标保留缺测状态，不以 0 或空记录表示健康正常。大字、键盘与窗口操作不依赖数据库。

此次整合未改写前端健康图表接口；2.0.6 既有验证及未验收范围见版本说明。修改限制在 `preview`，不随文档变更重建二进制或发布附件。
