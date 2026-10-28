# API v1 破坏性切换（2026-10-28）

## 边界

- 唯一对外业务入口：Go `/api/v1/`。彻底取消 `/restful/` 的旧路由、别名层、兼容代理以及前端调用。
- 会话 Cookie 由 `JSESSIONID` 改为 `SUINING_SESSION`。既有浏览器与手表控制端必须升级并重新登录；不会迁移旧会话标识。
- 前端 Vue 3 / Vite、桌面 Tauri / Rust、服务端 Go、MySQL 业务主库和 PostgreSQL 可选分析副库维持明确职责；没有 Java 运行时、Spring 或 Electron 构建链。
- 原生设备路由中的请求结构、签名及幂等条件不因 URL 切换而放松；涉及控制器固件或模拟器的旧路径，必须同步升级，否则将请求失败。
- 不删除历史 Release、数据迁移记录、过去的 ADR。旧数据库版本、已有业务数据不自动清除。

## 实施要求

1. 所有 Go `HandleFunc` 注册和 `accesspolicy` 的 CSRF/角色规则一并切换，新接口默认拒绝未授权访问。
2. 前端包括管理、家属、登录与照护页面全部改为新路径；Tauri 的管理员初始化和退出接口同样切换。
3. `scripts/generate-openapi.py --check` 要求提交的 OpenAPI 与 Go 当前路由精确一致，`internal/apidoc/aliases.json` 不再存在。
4. 构建检查：`go test ./...`、`go vet ./...`、`npm ci`、`npm test`、`npm run typecheck`、`npm run build`，并进行 MySQL/PG 环境下的真实集成测试。
5. 对外验证 `/restful/...` 为 404；`/api/v1/service/status` 返回主库连接状态；会话、设备、SOS、处理记录、通知和数据库降级行为必须通过回归。

## 部署影响

这是一个不向后兼容的协议升级，服务端和所有客户端必须协同发布，避免旧客户端不可用。下线旧路径并不意味着删除历史真实业务数据。MySQL 仍是唯一事务主库，PostgreSQL 只负责可选异步分析；网页和设备都不直接持有数据库连接凭据。
