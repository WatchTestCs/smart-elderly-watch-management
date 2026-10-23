# 连接级语句复用与延迟（2026-10-23）

基线 e39e14bf8500efbd74548ef012d244eaf8f320b1，接续参数插值版本，优先降低实际请求与事务的 p95/p99。历史报告的吞吐提升不能代替本轮延迟证明。

## 实现

OpenMySQL 的业务连接池使用包装过的 mysql Connector。每条物理连接维护最多 64 个原生 prepared statement（FIFO；MYSQL_STMT_CACHE_SIZE=0 可恢复前一版插值/临时 prepare，允许 0..128）。SELECT/INSERT/UPDATE/DELETE 的参数化模板复用服务器句柄，减少 SQL 文本反复解析、转义扩张与临时 prepare。没有跨连接复用句柄，没有另取池连接导致事务饥饿；database/sql 序列化物理连接操作，缓存不需要共享锁。PostgreSQL 和专用健康探测池保持原路径。

依赖已固定的 go-sql-driver/mysql 1.9.3 的公开 driver 接口，运行时核对完整接口集，原样转发参数转换、事务选项、取消、连接存活检查与池回收。显式 database/sql Prepare 获得独立语句，不能关闭缓存内的句柄。参数数量在执行之前校验；执行失败直接返回，不重放写入。只有 prepare 阶段 MySQL 1295（语句不支持 prepare）才回退原驱动路径。

无参数 Exec、非标准 DML（包括带前置注释的模板）、SET/USE/DDL 保守清理旧句柄后直通，以免保留旧 SQL 解析状态。只读 SHOW 和无参数 SELECT 不清缓存。连接关闭由 COM_QUIT 一次回收服务器句柄；FIFO 驱逐显式关闭旧句柄。

代价是每条连接持有服务器执行计划与模板，20 条业务连接默认最多 1280 个句柄，每副本叠加，须计入 MySQL max_prepared_stmt_count。冷启动及抖动时仍要 prepare/close，首次访问可能比插值多一次往返。缓存大小不是内存字节上限；不适合的部署可设 0 回退。没有修改刷盘、事务隔离、设备归属校验、owner/device 锁顺序和通知可靠性。

## 验证

单元测试覆盖完整键、重复复用、FIFO 上限、参数数量、错误不重放、取消与会话变更清理。隔离 MySQL 验证 session prepare 次数、JSON/BLOB/文本/微秒时间、事务回滚、FIFO 重准备与已取消请求不落库。已有传输模式与业务双库回归保持。

A/B 工作流在同一 runner、Go 1.23.2、GOMAXPROCS=2、MySQL 8.4 上交替三轮比较基线与候选：

- 原 SaveBatch+commit：每轮预热后 500×200 条，核对 100600 条历史。
- 新 HTTP：单次 1/200 条、1/4 个客户端，共四种配置；每种各三轮，每轮预热后 500 次请求。包含 loopback HTTP、JSON 解码、设备认证、事务/锁/commit 与响应读取；校验响应及持久化计数。

并发客户端向同一设备/owner 上报，覆盖真实锁排队；版本与 record identity 按客户端分离，不依靠回放去重制造速度。客户端 JSON 构造在请求延迟计时之外，但计入总运行与进程分配。此测试采用封闭客户端模型，不等于开放到达率排队或生产 SLA；不含全局中间件、代理与外网设备 RTT。冷启动、服务器资源、长时间及混合业务仍需部署环境测量。

本地短模式数据库用例跳过；精确提交的真实双库 CI、全部 A/B 数字及原始数据见 [PR #30](https://github.com/WatchTestCs/smart-elderly-watch-management/pull/30)。先确认延迟结果再决定保留方案，不预先声明加速。
