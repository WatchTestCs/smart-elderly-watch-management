# 提交前复审与修正（2026-10-25）

固定基线：`preview@daac58a9d20420cc4bdf71675be64bcc20cc8e49`。两路独立审查分别检查仓库规范与原始性能/兼容性要求；以下记录本地提交前验证，不替代后续远端 CI。

## Standards

发现两项文档问题，均已修正：

- 性能报告原先依赖未上传的本地附件和路径，违反 CONTRIBUTING 的可复现证据要求。现在提交完整四轮 HTTP 报告、CPU/分页原始输出、淘汰方案和解码器实验；增加可原样复制到固定基线的 `read_latency_ab_test.go`，在父目录 README 给出完整参数和命令。
- 时区/统计参考说明仍写着“等待验证”，与完成记录矛盾。已改为链接最终结果和保留的回退数据。

没有其他可操作的代码规范或结构性问题。许可证全文保留，文件职责索引同步更新。原有文档的 CRLF 约定保留；Go 格式按索引中的规范文本检查。

## Spec

未发现新的生产逻辑缺陷。检查了会话/MFA 与请求快照边界、分页精确总数和排序、NULL 哨兵、聚合指针所有权、日历与 DST 缓存、快速解码错误处理。

额外独立对照通过 12,006 组 Sample 接受行为、字段、标准序列化和浮点位比较；这是有限样本而非穷尽证明。新请求仍查询权威存储，单次请求内的已验证快照可复用。复审没有修改实测的生产 Go 或模块文件，[散列记录](evidence/production-source-hashes.json)可核对这一点。

## 修复既有测试竞态

重新执行完整短测复现 `TestRunnerDrainsSaturatedTransactionalBacklog` 失败。固定基线三次中有两次同样失败：PG 已有6,000条，而最后250条 MySQL outbox 确认尚未完成。该观察窗口符合先提交 PG、随后确认 MySQL 的生产顺序。

修正仅在测试：继续轮询，直到 PG 恰有6,000条且 MySQL outbox 为0，才记录完成耗时和检查 Configured。保留原20秒超时、数量和配置断言，没有调整生产事务顺序。修正后连续5次通过，Go1.23 analytics 全包重跑通过，Go1.27 race 中 analytics 全包通过。

## 执行验证

本轮使用独立本机 MySQL8.0.46 和 WSL PostgreSQL16；之前的性能实验/集成验证使用 MySQL8.4。没有使用生产数据。新增测量入口的元数据改为 caller variant/VCS build settings，避免把固定基线字段当作候选提交号；显式开启时缺少 MySQL DSN 会失败。

- Windows Go1.23.2 完整短测执行：1,032个测试/子测试通过，10个测试跳过，唯一失败为上述竞态；修正后 analytics 全包和专项5轮通过。其他包原完整执行通过。
- Windows Go1.27.2 重点兼容性测试通过；Go1.23.2/1.27.2 的 `go build ./...`、`go vet ./...` 均通过。
- Linux Go1.27.2 执行 `go test -race -short -p=2 ./... -count=1 -timeout=10m`。首轮两例测试在初始化测试数据库连接时报告5秒 `MySQL unavailable`，其余包通过；这不是业务断言失败。相应 MFA/夜间活动测试使用 `-race -p=1 -count=3` 串行重跑，全部断言通过。保留首轮失败，不描述为单次全量全绿。
- 真实 PostgreSQL16 的分页契约回归通过；`go mod verify` 和 OpenAPI生成契约检查通过，API文档语义未变化。
- opt-in 性能/耐久测试的默认跳过不计为运行通过；本次未重新测量生产端到端 SLA 或执行长时间耐久测试。

精选原始输出与命令状态见 [review-validation.txt](evidence/review-validation.txt)。原始性能实验的全部方向和局限仍保留在父目录报告中。

复审汇总：Standards 两项已修复；Spec 无新增生产行为问题；另修正一项已在固定基线复现的测试时序竞态。
