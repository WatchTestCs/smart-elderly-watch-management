# Runbook: Rollback

桌面应用没有“自动数据库回滚”机制。遇到发布问题时优先回滚程序二进制，而不是直接修改/删除用户数据库。

1. 停止发布扩散，不移动已有 tag。
2. 保留失败版本的日志、release manifest 与 hash。
3. 判断问题是否涉及 schema/data mutation。
4. 若只涉及程序文件，重新安装已知良好的旧版本，并保持 `%APPDATA%\岁宁` 不变。
5. 若涉及 schema 变化，在恢复数据前先复制整个用户数据目录。
6. 不在未知数据库状态下反复启动不同版本。
7. 发布新的修复/回滚版本，记录 supersede 关系。

任何未来破坏性 schema migration 都必须在对应 ADR 中定义 forward/rollback 策略。
