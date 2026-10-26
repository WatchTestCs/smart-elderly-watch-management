# 验收矩阵

本机完整进程演练：`110b5a43b400f6018c496855b1458d47181785aa`，
`evidence/reliability-20261026/process-matrix-110b5a4/matrix.json` 的 complete=true。
每次使用全新 UUID 测试库、真实 MySQL/PG、TLS gateway、Prometheus、Alertmanager、独立 receiver
和 watchdog 进程；正式规则的 for、2 分钟租约、5/10 分钟升级保持真实时长。
receiver_accepted 与人工响应分别记录。没有手工改任务状态。

| 场景 | 结果/证据 |
|---|---|
| 重复设备 requestId | 同事件 ID；MySQL 已提交；没有重复 incident |
| 签名回执与重复回执 | delivered、审计幂等；incident 仍无人接单 |
| 网关接收后断开 TLS 连接 | 实际网络断开，稍后签名回执收敛 delivered，attempts=1；不会被旧失败覆盖 |
| PG 真进程停止 + 网关持续 503 | MySQL/SOS 保持就绪，analytics warning/queue critical 送到独立 receiver，无人接单按期 escalated |
| PG 恢复 | 自行连接，持久 analytics backlog 重放至 0 |
| 持续 429 | failed 保留；8 秒内没有追加尝试，按实际 30 秒退避再次尝试，记录 UTC retry 时间 |
| 网关进程停止 | network 分类；任务保留，receiver 仍活着 |
| 无效 DNS 域名 | 独立 dns 分类，不混为 HTTP 错误 |
| worker 发送中强制 kill | provider 已接受后终止 app，正常重启；等实际 2 分钟 lease，attempts≥2，provider 接受仅一次，最终 delivered |
| MySQL 真停库 | databaseReady=false，SOS HTTP 503；healthz 仍 200 但不当作业务健康；critical 到 receiver |
| MySQL 真恢复 | 不改数据库任务状态，服务和新 SOS 自行恢复 |
| heartbeat 被行锁阻塞 150 秒 | 先确认上一故障心跳恢复/旧告警解除；本次新 startsAt 的 stale critical 到 receiver，业务 DB 仍 ready；释放后持久心跳恢复 |
| acknowledge 后无人处理 | 原始 audit 显示 acknowledge 后 614.111 秒 escalated，满足原 10 分钟 deadline；acknowledged_by 保留且未 resolved，见 final-audit-timeline.json |
| app 强制终止 | Prometheus/Alertmanager/receiver 继续运行，SuiningServiceDown 实际收到 |
| Prometheus 强制终止 | 另一 watchdog/receiver 继续运行，PrometheusUnavailable 实际收到 |

其他绑定当前 CI 源 `f1417a9` 的强制回归及专项证据：

| 场景 | 结果/原始记录 |
|---|---|
| 真实 HTTPS 请求超时，完整 SOS→receipt→ack→处理→resolve→另一个 actor verify | TestCareFullHTTPChainAfterRealGatewayTimeout；12 秒实际超时；MySQL 状态与 audit 核对，拒绝同人 verify |
| 旧 worker 覆盖/晚回执/重复审计/全局 ADMIN duty | care/handlers 强制回归通过；原始 mysql.jsonl/postgres.jsonl |
| delivered 仍未接手的 UI | frontend 21 tests 通过，含相应显示断言；未执行真实人工救援 |
| 双实例共享图片，A 下线 B 继续读 | TestImageS3TwoInstancesAndRealRestore，真实私有 S3、独立 DB session；匿名 GET 拒绝 |
| 真实备份后删除测试 DB+blob 再恢复 | 同测试：真实 mysqldump/mysql，哈希一致、全部引用可读、SOS/outbox/session 保留；不把空库当成功 |
| GC/引用竞争/备份租约/补偿/篡改 | blob 单元及真实 MySQL 回归通过；孤儿宽限与 deleting 墓碑保护 |
| 100000 来源、256 workers、IPv4/IPv6、NAT、伪造代理 | ratelimit race 回归通过；另保存 300000 微压测 RSS；不是 HTTP 吞吐 |
| 跨实例登录审计与 IP 轮换 | 两 Handler 共享 DB 回归通过，trusted proxy 与审计 IP 一致 |
| key A→B→重封装→去 A，错/缺 key | securestore 回归通过；坐标错 key 不泄露密文；真实旧数据库批量迁移仍待验 |
| 全套 CI | 两版本各 MySQL suite 1106 pass/10 opt-in skip，PG-only 825 pass/42 可解释 skip，build/vet/OpenAPI/前端/Windows/安全成功 |
| 两历史 exact SHA | 50b9cc0=1041 pass、41b0f1d=1068 pass，各 10 opt-in skip；真实 MySQL/PG/race，build/vet 成功 |
| 发布失败关闭 | 缺失/过期/无 hash artifact、旧 SHA、分支移动、最新失败不能用旧成功覆盖，5 项测试通过 |
| watchdog 交付失败后重启与恢复顺序 | 真实 HTTP 503、持久 SQLite、恢复后先 firing 再 resolved，2 项测试通过 |

未验收：真实设备/供应商/人类、宿主机断电与异地寻呼、生产规模恢复/长测、历史全库加密迁移、
管理员保护与签名发布。所有数字仅代表各文件标明的环境及 SHA，后续分支 HEAD 必须单独复验。
