# 2.0.2 维护交接

入口为 README 与 ARCHITECTURE。服务为 backend，桌面为 desktop 的 Rust/Tauri，前端为 oldIot。构建使用 desktop/build.ps1，版本核对使用 validate-version.ps1，签名使用 sign.ps1。

数据库离线时先确认窗口能打开，再检查外部 PostgreSQL、部署变量、网络与迁移权限。`/healthz` 是进程存活，`/restful/service/status` 的 databaseReady 是业务数据库就绪；两者不能混用。

旧数据库不会自动迁移；用户数据目录仍保留旧资料。不要删除旧 H2 文件、清空设备归属或修改生产库来让迁移“看似通过”。正式签名证书和生产数据迁移仍需部署方安排。

代码只提交 preview，当前发布 2.0.2-preview.1。历史审查材料要明确标注历史，不得挪用旧测试结果证明新桌面壳。
