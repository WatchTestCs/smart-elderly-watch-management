# 岁宁 Go 后端 · 2.0.2

由 Tauri 管理生命周期的本地 Go 服务，负责业务 API、会话、设备接入、图片和内嵌前端。业务数据位于独立 PostgreSQL，软件打开与本地 HTTP 服务不等待数据库。

## 运行与配置

源码开发使用 Go 1.27.2；桌面用户无需安装 Go。完整安装包请从仓库 Releases 获取。

```sh
export DATABASE_URL="postgres://用户:密码@127.0.0.1:5432/数据库?sslmode=disable"
export DESKTOP_TOKEN="开发环境专用的随机令牌"
go run ./cmd/server
```

仅在受信任的本机开发环境使用上述非 TLS 示例；跨主机数据库连接应验证 TLS。`ADDR` 默认 `:8080`，`IMAGE_DIR` 默认 `./data/images`，`SESSION_TTL_HOURS` 默认 12。生产桌面由 Tauri 传入仅回环的动态监听地址、上传目录、随机令牌和 `SUINING_DATABASE_URL` 对应的连接串。不要把连接串提交到 Git。

没有默认管理员账号。首次初始化仅接受带桌面令牌的请求；Tauri 通过受限原生命令完成初始化，令牌不暴露给页面。普通写接口使用会话与 CSRF，设备接口使用设备鉴权。日志不应包含密码或连接串。

## 源码职责

- `cmd/server`：HTTP 生命周期、数据库后台恢复、迁移和父进程监测。
- `cmd/migrate`、`migrations`：独立迁移入口与编译内嵌 SQL。
- `internal/db`：pgx 连接池与迁移串行化。
- `internal/auth`、`session`、`csrf`、`ratelimit`：角色、BCrypt、会话、请求防护。
- `internal/handlers`：业务与服务状态接口、图片校验、静态资源。
- `VERSION`：与桌面、前端保持一致的产品版本。

前端发布构建复制到 `internal/handlers/web` 后由 Go 编译内嵌；源码只保留占位文件。不要把未嵌入前端的纯 API 二进制当成桌面交付物。完整构建和安装说明见仓库 `docs/BUILD.md`、`docs/SETUP.md`。

## 迁移与存量数据

迁移 SQL 由 migrations/embed.go 编入程序。server 和 migrate 命令均无需从 backend 目录启动，也不依赖磁盘上的 migrations 目录。迁移通过事务级 advisory lock 串行执行，已应用版本会跳过。

0004 增加 device.old_id 到 oldperson.old_id 的外键，禁止删除仍绑定设备的长者。若旧库已有孤儿设备引用，迁移会明确失败并回滚该版本，不会删除设备、历史记录或静默清空归属。升级前备份数据库并检查：

```sql
SELECT d.device_id, d.old_id FROM device d
LEFT JOIN oldperson o ON o.old_id=d.old_id
WHERE d.old_id IS NOT NULL AND o.old_id IS NULL;
```

若查到记录，请由管理员确认设备的实际归属后修复，再重试迁移。生产连接账号需要迁移相关 DDL 权限；不支持把旧 Java/H2 文件直接当成 PostgreSQL 数据库使用。

## 本地回归测试

```bash
go test ./...
# 使用独立测试数据库；测试为自己创建的临时 schema 建表，结束后仅清理该 schema。
TEST_DATABASE_URL='postgres://用户:密码@127.0.0.1:5432/专用测试库?sslmode=disable' go test -race -count=1 -timeout 120s ./...
go vet ./...
```

未设置 TEST_DATABASE_URL 时，数据库测试明确显示 SKIP，不能称为集成验收通过。数据库账号需有建 schema 权限。浏览器 QA 拦截 API，因此不能替代这些真实数据库测试。

仅隐藏占位文件时，后端可按纯 API 模式编译；桌面发布仍必须先运行前端构建并嵌入完整静态资源，不能把纯 API 二进制用于桌面交付。

## 数据库离线启动

HTTP 监听、桌面就绪文件、前端资源与退出接口不等待数据库。连接和迁移在后台执行；连接失败、配置错误或迁移失败不会终止界面。每次连接探测限时 5 秒，成功连接后的迁移独立限时 2 分钟，失败后后台重试。

GET /restful/service/status 返回 databaseReady，不返回连接串或数据库错误详情；数据库未准备好时，业务 API 返回 HTTP 503、DATABASE_UNAVAILABLE 与 Retry-After。前端展示连接状态页，恢复后重新检查会话并继续。失败的写入不会被自动重放。

GET /healthz 仅表示本地进程存活，不表示数据库就绪。数据库和本地软件仍独立部署，不提供离线假数据或未连接时的本地业务写入。
