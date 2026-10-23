# 健康上报的窄字段读取（2026-10-23）

基线 60be581e69048c6e9f3be680fcbee3067b8750e8。事务外准备实验复测不划算，已撤回其生产改动；[全部 72 轮数据与代价](../2026-10-23-preparation/results.json)保留。本轮不再更改事务边界、连接借用次数或内存生命周期。

健康上报此前在认证及 device 行锁读取时各扫描 22 个字段，包含 UI 资料、坐标、电量、应用版本和处理计数；实际只读取 6 个业务字段。现在两次查询（以及实验 late-lock 的最终查询）只投影 device_id、device_no、secret、old_id、status、health_epoch。仍使用相同设备号查找和主键 FOR UPDATE，保留新鲜绑定/密钥/状态复核。全字段与窄字段认证共用同一校验函数，避免两套授权策略分化。

健康上报从不提供 battery 或 appVersion，因此用专门 UPDATE 只写 last_time 与 last_online_at；两者传相同 now，使用顺序 $1/$2/$3 参数，避免不必要的 COALESCE、NULL 参数与重复参数重排。其他设备接口保留完整资料与通用更新函数。没有更改 owner/device 锁顺序、提交确认、历史、刷盘或缓存容量。

预期减少结果集字节、字段转换和参数分配；数据库仍须读取聚簇行，不能把 22→6 字段解释成数据库 IO 或总延迟按比例下降。事务 SaveBatch 控制组的生产代码恢复为 60be581；不能把控制组计时变化归为这次 handler 优化。

保留已有来源/权限/绑定、并发轮换/停用/解绑、冲突回滚、历史和单连接池回归。五种 HTTP 配置与直接事务控制组仍在同 runner、MySQL 8.4、Go 1.23.2、GOMAXPROCS=2 上交替三轮比较 60be581，夹具一致、全部轮次核对响应与数量。本地 race-short/build/vet 的数据库用例跳过，真实 CI、原始结果与回退见 [PR #30](https://github.com/WatchTestCs/smart-elderly-watch-management/pull/30)。
