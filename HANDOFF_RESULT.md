# 可靠性修复与本机验收结果 · 2026-10-26

按用户要求先提交计划，再集成/修复，再从失败中微调验收。只更新新分支
`codex/reliability-validation-20261026`；未修改 preview/main、原 PR 分支、仓库保护或生产环境。
本次执行范围由用户明确为本机隔离环境，真实渠道和人工值班条件未提供。

## 结论与版本

- 原交接基线 `50b9cc094002ca691244b7d504cdeedf117787dd` 已在真实 MySQL 8.0.43 + PG 16.15
  下通过 race 全套：1041 pass、10 可选性能实验 skip，build/vet 成功。最初 HDD 超时失败也保留。
- 实际开始时 preview 已为 `41b0f1d8752d1cf023872b067af8655f7a2580fd`，已有 exact-head CI、
  Security、20 分钟 endurance 成功；交接中的“最新无 CI”是旧状态。该版本本机 1068 pass。
- 功能和完整进程演练冻结于 `110b5a43b400f6018c496855b1458d47181785aa`；随后只增加 CI 来源
  加固、文档和证据。当前实现与这些验收版本的对应关系保存在每份 manifest/source-sha 中。
- `f1417a9121eed4036f5001250d9703d54520ca75` 的 CI 和 Security 全部成功。两个 Go 版本各
  1106 主套件 pass/10 opt-in skip，PG 专项 825 pass/42 已解释 skip。真实 CI MySQL 8.4.13、
  PG 16.15，durability 开启；S3 双实例和真正删除后恢复在 mandatory suite 内成功。
  [CI run](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/40311522133)。
- 报告/证据本身后续提交产生的新 HEAD 仍须重新通过 CI；最终状态见新分支最新 Actions 及
  release-gate.py 输出。上面各 SHA 的成功不能作为未来提交的发布许可。

## 任务结果

| 任务 | 完成内容 | 验收边界 |
|---|---|---|
| TASK-01 发布门槛 | exact-source CI、真 MySQL/PG/race、migration/reapply/rollback、build/vet、OpenAPI、前端、Windows sidecar、安全与 SHA256 artifacts；两个手动发布入口受门槛约束 | 完整 Windows installer 在合入经审查 preview 后手动执行；不冒充已签名发布 |
| TASK-02 SOS | 集成 #27；重复/晚到回执、旧 worker CAS、ADMIN duty 全局约束；真实 TLS 超时完整 HTTP 状态链与双人复核回归 | 合成网关/actor，不能代替真实渠道、真人救援 |
| TASK-03 故障/恢复 | 集成 #29；真实停 PG/MySQL、kill app/worker、heartbeat 行锁、429/5xx/DNS/连接断开、独立 Prometheus/AM/receiver/watchdog；真实 DB+blob 删除恢复 | 同机进程隔离通过；整机故障、异地接收及生产体量 RPO/RTO 待验 |
| TASK-04 滥用控制 | 全进程 key/槽硬预算、有限过期回收、可信代理、边缘共享限流、跨实例登录审计；30 万来源/256 并发/RSS 和 NAT/伪造 header 测试 | 多 ingress 仍需上游统一配额；不宣称进程 limiter 是分布式全局额度 |
| TASK-05 静态保护 | 基于 #32 的字段保护；385 个 MySQL 字段、PG 投影及资产分类；版本 keyring、独立 lookupKey、A→B→移除 A 和 fail-closed 回归 | 分类/迁移设计已交付；旧明文、旧 HMAC 索引、真实历史密钥迁移尚未自动执行 |
| TASK-06 图片 | local/S3 抽象、AEAD、哈希、不可变写、补偿、墓碑/引用锁 GC、备份租约、双实例/真恢复 | 私有真实 S3 服务通过；旧本地图片切 S3 需受控迁移，不能只改环境变量 |
| TASK-07 治理 | 两入口固定 SHA/证据校验、来源清单与哈希；可导入 ruleset 提案 | 管理员保护、独立 reviewer、签名 tag/证书尚未应用，不伪造签名或批准 |

## 集成边界与修复

以当前 preview 集成原 PR #27、#29，处理 limiter 所属生命周期和分析投影提交确认冲突，保留
事务/审计语义；#33 CI annotation 已兼容集成。#32 仅采用三个安全提交
`c709daf`、`32fc08b`、`ac1fceb`，并迁入 Secure cookie；其后续跨请求 session 缓存会延迟
跨实例撤销，未整支合入。保留每请求读取数据库的会话撤销语义，不另造相同功能。
迁移 MySQL 0013/0014 避开 #32 预留的 0012，PG 为 0011/0012。

新增网关 TLS CA 配置和严格验证、禁止 redirect、12 秒超时、有限错误分类与 Retry-After 退避；
错误记录不含网关 URL/token。坐标错 key 读取失败，不向客户端返回密文。增加队列年龄/采样有效性
及 TLS/网关告警。网关成功仍仅为 sent，签名 receipt 才可进入 delivered；人工状态独立。

保留了三轮实际失败：HDD DDL 超时；新增 CI 的回滚注释/迁移计数/Linux TCP 客户端问题；首轮
故障脚本等待不足及旧 firing 告警归属问题。修复后重新验证，没有删除测试或缩短业务 deadline。

## 阅读交付物

- [逐场景验收矩阵](validation-matrix.md)、[原始证据索引](evidence/reliability-20261026/README.md)。
- [完整分类与轮换](docs/reliability/data-protection.md)、[图片与协调备份](docs/reliability/storage-operations.md)。
- [滥用控制/独立监控](docs/reliability/abuse-and-monitoring.md)、[本机性能/RSS](docs/reliability/performance.md)。
- [环境/重现/回滚/生产门槛](docs/reliability/environment-and-rollback.md)、[仓库治理提案](docs/reliability/release-governance.md)。

尚未通过的生产门槛不能由本机绿灯替代：真实手表/渠道送达、独立值班人员响应、异机/整机故障、
历史明文与密钥全量迁移、最旧备份恢复、生产容量长测、多 ingress 统一限额、管理员规则及独立
review、签名发布。当前交付可在新分支审查和复验，不据此宣布已满足生产上线条件。
