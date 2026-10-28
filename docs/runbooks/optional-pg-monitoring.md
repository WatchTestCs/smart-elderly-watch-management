# PostgreSQL 可选分析与独立告警验证

> 适用于 `deploy/platform.compose.yml`。这里的数据库高可用边界不变：MySQL 是唯一业务主库，PostgreSQL 只存储可重建的异步分析投影。PG 不是 MySQL 的热备或自动容灾实例。

## 修复的启动依赖

- `app` 只等待 MySQL 服务健康，不再等待可选 PG：即使 PG 离线，Go 应用也可以启动并接受核心业务操作。
- `prometheus` 不再等待 app 健康，应用停机或断库时监控服务仍可启动；`gateway` 保留 app 健康依赖，不对外转发未就绪业务。
- 后端在已有的受 `METRICS_TOKEN` 保护的 `/metrics` 中提供 `suining_analytics_configured`、`suining_analytics_ready`、`suining_analytics_outbox_pending` 和 `suining_analytics_outbox_sample_valid`。队列计数约每 30 秒采样，取样失败时以 `sample_valid=0` 表示不可用，不能用旧值作为实时队列。
- 每次通知后台将心跳**成功写入业务库之后**更新 `suining_care_worker_last_heartbeat_unixtime`；心跳仅能证明工作循环与数据库写入最近成功，不代表供应商已送达或值班人员已出发。
- `suining_care_notification_configured` 是网关 URL + 密钥的配置状态，不是渠道在线或实际送达指标。

## 配置校验与故障注入（仅隔离测试库）

在仓库根目录准备 `scripts/start-platform.py` 产生的环境配置。不要在有真实老人档案的库做中断演练。

```sh
docker compose --env-file deploy/platform.env -f deploy/platform.compose.yml config --quiet
docker run --rm -v "$PWD/deploy:/etc/prometheus:ro" \
  --entrypoint promtool prom/prometheus:v3.5.0 \
  check rules /etc/prometheus/alerts.yml
docker compose --env-file deploy/platform.env -f deploy/platform.compose.yml up -d
```

建议在单独测试环境执行：
1. 暂停 PG：`docker compose --env-file deploy/platform.env -f deploy/platform.compose.yml stop postgres`。
2. 检查 app 继续运行、`GET /api/v1/service/status` 返回 `databaseReady=true`，同时分析投影 `ready=false`。留意 worker 心跳、通知消费和 MySQL 存储仍持续成功。
3. 访问经授权的 `/metrics`、Prometheus `/alerts`，核对 PG 不可用仅触发 warning，不将 MySQL 主库判定为故障。
4. 重启 PG，观察 outbox 同步逐步减少。队列丢失或 PG 表丢失时使用 `docs/DUAL_DATABASES.md` 的重建流程，而不是从 PG 恢复主业务。
5. 单独停止 app，确认 Prometheus 仍运行，触发 `SuiningServiceDown` 或 `SuiningScrapeMissing`；恢复后检查 SOS 排队与有无值班人员接单。

**注意**：此处只提供 Prometheus 告警规则，未配置 Alertmanager / 第三方通知接收器，**不会自动发短信、打电话或通知值班人员**。生产需要独立、经演练的值守告警通道、UPS/网络冗余、MySQL 备份和恢复演练，以及明确的线下应急接管路径。

## 告警策略边界

- `SuiningCareWorkerNeverReported` 和 `SuiningCareWorkerStale` 表示没有可靠的持续任务处理证据，属于 critical。
- `SuiningOptionalAnalyticsUnavailable`、`SuiningAnalyticsBacklogGrowing` 属于 warning；PG 故障不应使 SOS 持久写入失败，但积压需要限制和监控磁盘。
- 未配置照护通知网关会触发 warning；已配置网关但缺失长度合规的 `CARE_RECEIPT_SECRET` 会触发 critical，因为发出消息后无法确认送达。两者都不能替代真实网关的签名回执/接单验收。
- 指标在 scrape 周期和 worker 写入周期之间存在延迟；告警阈值为起点，须按机构容量、实际 p99 和值班要求调整。
