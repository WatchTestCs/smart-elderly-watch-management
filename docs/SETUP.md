# 环境配置与初始化指南（MySQL 主库 + PostgreSQL 分析库）

当前版本采用“MySQL 业务主库 + PostgreSQL 异步分析库”架构，完整的容器编排与旧库迁移细节请参阅 [双库配置指南](DUAL_DATABASES.md)。

## 部署与初始化步骤

1. 准备一个空的 MySQL 数据库用于存放业务数据，并按需准备独立的 PostgreSQL 数据库用于异步分析。
2. 配置环境变量：直接运行 Go 服务端时设置 `MYSQL_DSN` 与 `PG_ANALYTICS_URL`；启动 Windows 桌面端时设置 `SUINING_MYSQL_DSN` 与 `SUINING_PG_ANALYTICS_URL`，同时配置 `DATA_ENCRYPTION_KEY`（或 `DATA_ENCRYPTION_KEY_FILE`）。修改环境变量后需重启程序。
3. 首次启动后，Go 服务会自动在 MySQL 中执行表结构迁移。迁移完成后，通过桌面端引导页面创建首位管理员账号（系统不设默认账号或通用密码）。
4. 登录管理后台后，先在“老人档案”中创建长者资料，再前往“设备管理”登记并绑定手表。编辑长者基础资料不会改变已绑定设备的归属关系；若移除主设备，系统会自动将剩余已绑设备中编号最小的一台设为主设备。
5. 当收到手表上报的 SOS 求助或异常告警时，照护人员可在工作台接单并登记处置措施，完成后由另一名主管独立复核并关闭告警。护理员与审计员账号仅开放照护协作相关页面。
6. 主管可在“值守诊断”页面查看 PostgreSQL 分析库的连接状态、待同步事件积压量以及最近同步时间。外部通知网关与回执配置方式见 [照护服务部署](CARE_SERVICE_DEPLOYMENT.md)。

## 故障恢复与旧库导入

当 MySQL 临时离线时，客户端仍能正常打开界面并保留已填写的表单内容，但业务保存请求会返回 `503` 提示手动重试，系统不会在后台自动重放失败的写请求。PostgreSQL 离线则仅推迟分析数据同步，不影响日常业务保存。

如需从早期使用 PostgreSQL 作为业务主库的版本迁移数据，请先停止新旧服务写入并备份数据库与图片目录，随后设置 `LEGACY_POSTGRES_URL` 与目标 `MYSQL_DSN`，执行以下命令：

```powershell
# 预检查源库与目标库状态（只读）
go run ./cmd/transfer-postgres
# 将数据单事务导入空的 MySQL 数据库
go run ./cmd/transfer-postgres --apply
```

导入完成后，请将旧版图片目录手动复制到新环境的图片目录下（Windows 桌面端默认存放路径为 `%APPDATA%\岁宁`）。
