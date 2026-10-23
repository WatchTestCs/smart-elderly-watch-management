# 条件读取旧 JSON 与事务延迟（2026-10-23）

基线 `93d609b10ec44b7245cfcde3173bb8da7e86512e`。继续降低健康上报的持锁时间和 p95/p99；[上一轮全部实验](previous-pr30.md)保留，包含尾延迟回退。

## 减少每个事务的一次往返

业务连接建立时显式设置 session READ COMMITTED。连接已知仍保持该默认值时，BeginTx 的 READ COMMITTED 请求转发为驱动默认级别，省去重复 SET TRANSACTION，保留 BEGIN 和真实 COMMIT。其他显式隔离级别与只读选项仍按原请求传给驱动。SQL 会话变更、CALL、未知语句以及对它们的显式 Prepare 都永久撤销该连接的已知默认状态，之后恢复每事务显式设置；没有根据请求参数猜测当前隔离。普通迁移注释可识别，MySQL 可执行版本注释保守撤销。SQL() 的裸连接默认也变为 READ COMMITTED；需要其他默认隔离的内部工具须显式指定。

用隔离 MySQL 的 Com_set_option 验证省去往返，并把 session 改为 REPEATABLE READ 后通过第二连接提交更新，确认指定 READ COMMITTED 的事务能看到新值，防止只看 session 变量而误判实际事务隔离。设 MYSQL_STMT_CACHE_SIZE=0 会恢复原事务设置路径和原参数传输路径，但连接初始化仍使用统一 READ COMMITTED 默认。

## 实现与语义

旧路径无条件读取每个已有样本的整份 JSON；版本始终递增的批次也必须把旧 JSON 从 MySQL 取回、复制、传输，然后直接丢弃。现在计算输入批次的最小版本，用 SQL CASE 仅对 `stored.version >= minimum_input_version` 读取 payload，较低版本只返回 NULL；仍然读取身份、版本和修订号。此条件同时兼容 MySQL 和 PostgreSQL，不增加查询往返。

证明：若旧版本小于输入的最小版本，该身份的每个输入版本都更高，旧 JSON 不可能参与同版本冲突比较。只要可能相等，完整 JSON 就仍会返回。对乱序、同批多次修改、混合版本、旧版本重放、删除、规范化比较和冲突回滚，原算法及事务保持不变。全批最小版本是保守边界：版本混杂时仍可能读取不需要的 JSON，宁可少省而不遗漏冲突。没有基于历史缓存猜测新数据，也不跳过数据库读取或归属校验。

两次 INSERT 共用事务私有参数数组；第一个 Exec 同步消费完参数后重新填充，避免第二个大指针切片。不进入全局池、不跨请求共享或复用 Sample/JSON 内存。

## 验证与测量

随机等价测试保留原值复制参考实现，扩展较高输入版本并裁剪不可能比较的旧 payload，仍核对完整状态、重复数、顺序、历史及错误。真实 MySQL 验证 NULL 裁剪、必须保留的 JSON、大 metadata、混合更新/重放，以及同版本冲突后整个事务回滚。

同 runner 的 A/B 基线为 93d609b：

- 500×200 条 SaveBatch+commit，新增 begin/save/commit 的 p50/p99 和最慢五次事务分段数据，保留全部墙钟延迟。
- 1/200 条、1/4 个客户端的 HTTP 各三轮×500 请求。
- 额外每样本 1024 字节 metadata、200 条×4 客户端，各三轮×200 请求，观察大 JSON 工作集。原样复制夹具到基线，仍核对响应与持久化数量。

串行/封闭客户端、loopback HTTP 与 CI runner 不能代表生产 SLA；外网、代理、真实设备和开放到达率仍不在测试内。阶段 p99 不能相加，最慢事务分段供判断具体等待来源。最终精确 SHA 的双库 CI、所有轮次与失败/回退见 [PR #30](https://github.com/WatchTestCs/smart-elderly-watch-management/pull/30)。不更改刷盘、隔离级别、数据保留或成功确认时机。
