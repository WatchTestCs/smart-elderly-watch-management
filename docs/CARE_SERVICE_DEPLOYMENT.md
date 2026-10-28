# 独立部署照护后台（2.0.4）

此部署方式使 Go 后端与桌面 GUI 的退出独立。**单实例并不等于真正高可用**：硬件接入、短信运营商、数据库、网络、供电和现场救援均需要另行保障。

当前本地双库配置、旧库迁移与 PG 故障边界见 [DUAL_DATABASES](DUAL_DATABASES.md)。

## 运行模式

- 在 Linux 服务器持续运行 `backend/cmd/server` 编译的 Go 程序，Windows 桌面端只作为客户端使用，不启动/终止这个服务端进程。
- MySQL 主库与 PG 分析库需分别持久化，业务备份以 MySQL 为准；所有数据库迁移在启动前由服务执行，首次升级前必须备份。
- `/healthz` 仅表示 HTTP 服务可连接；`GET /api/v1/service/status` 的 `databaseReady` 字段用于 DB 就绪诊断。注意 Web 和通知网关可用性要另行监控，不能把 HTTP 200 当作 SOS 可被救援的证明。

## systemd 部署示例

发行源码包含 `deploy/suining-care.service`，修改路径、建专用最小权限账号后：

```sh
sudo install -d -m 700 -o suining -g suining /var/lib/suining
sudo install -d -m 700 /etc/suining
sudo install -m 600 /dev/null /etc/suining/suining.env
sudo install -m 644 deploy/suining-care.service /etc/systemd/system/suining-care.service
sudo systemctl daemon-reload
sudo systemctl enable --now suining-care
sudo systemctl status suining-care
```

**环境文件示例（仅展示键，不要提交真实密钥）：**

```env
ADDR=127.0.0.1:8080
MYSQL_DSN=suining:REPLACE@tcp(127.0.0.1:3306)/suining?tls=true
PG_ANALYTICS_URL=postgres://suining_analytics:REPLACE@127.0.0.1:5432/suining_analytics?sslmode=require
IMAGE_DIR=/var/lib/suining/images
CARE_NOTIFY_WEBHOOK_URL=https://your-trusted-gateway.example/api/notify
CARE_NOTIFY_WEBHOOK_TOKEN=REPLACE_WITH_SECRET
CARE_RECEIPT_SECRET=REPLACE_WITH_RANDOM_SECRET_AT_LEAST_32_CHARACTERS
CARE_ESCALATION_PHONE=REPLACE_WITH_ON_CALL_PHONE
```

API 不应直接暴露到公网，建议用可信 TLS 反向代理和网络 ACL。以上 `ADDR` 地址与数据库连接字符串须按实际部署更改；若服务跨机器供设备接入，监听地址及公网域名/防火墙等也须独立配置。

## 通知状态与网关契约

SOS 成功响应是 `received`，不是 `delivered` 或 `acknowledged`。若有已绑定家属手机号，将在同一事务生成状态 `pending` 的 outbox 记录。未配置网关时任务安全留在数据库等待部署人员处理。后台每约五秒领取待发送任务，带有 `Idempotency-Key: care-outbox-<notificationId>` 的 POST JSON 请求；**网关必须持久化该键并对重复发送去重**。HTTPS 2xx 表示网关已接收（`sent`），不是已送达。

网关确认真实送达后必须带以下请求调用 `POST /api/v1/care/receipt`：
- JSON：`{"notificationId":123,"status":"delivered"}`
- `X-Care-Timestamp`：当前 Unix 秒
- `X-Care-Signature`：HMAC-SHA256 十六进制，密钥 `CARE_RECEIPT_SECRET`，消息为 `timestamp + "." + 原始 JSON 字节`

该接口拒绝无签名、过期签名、未知或尚未尝试发送的通知；对已标记 `delivered` 的**合法签名重复回执**返回 200，但不会再次写入送达事件。`sending`、`sent` 以及**有实际尝试次数的 `failed`** 均可凭可信签名首次确认送达（后者对应超时但网关随后送达的场景）。如果超过 5 分钟时间窗，网关重试时必须重新生成时间戳和 HMAC，而不能原样重放过期签名。生产网关需要独立端到端验收；供应商 HTTP 2xx 仍不能证明“家属已收到”。

## 人工处置和升级

管理员使用 `/api/v1/care/incidents` 查询、`/api/v1/care/transition` 接单/开始处置/登记措施/复核；每个动作均形成审计事件。超过当前时限而未完成时，服务将写入 `escalated` 并对 `CARE_ESCALATION_PHONE` 创建额外的待发通知（如已配置）。超时升级是软件流程，仍不等于救援人员已到场。

应对未完成任务设置独立可观测性：定时统计超过期限、未接单、网关发送失败和服务下线的任务，由值班团队在外部渠道进行确认。对生命安全事件，应保留电话/人工备用求助方式。

## 运维与恢复检查

至少按机构风险与 RPO/RTO 制定 MySQL 业务库及上传图片的备份计划，并在隔离环境定期演练 MySQL 恢复及图片关联校验；PG 分析库只作为可重建投影，不能替代业务备份。保留校验和、恢复时间目标、加密密钥保护及备份删除策略，详见 [备份与恢复手册](runbooks/backup-restore.md)。监控数据库连接、网关退避队列长度、最长未接单时间、告警升级数量及服务退出次数。

当前仓库不能证明真实硬件、短信运营商、应急人工值守和多机故障转移通过；这些都是正式交付的门槛。

## 2.0.5 值守健康检查与恢复

可在主机上执行 `sh deploy/check-care-health.sh`（需要 curl、Python 3 和可选的 systemd），以只读方式检查常驻服务、HTTP 连通性和数据库就绪。该探测**不代表通知工作进程、短信网关或值班人员正常**。

主管登录工作台后可查看 `care_worker_heartbeat` 状态：心跳超过 30 秒会突出警告，同时展示待发送、失败、发送中断和超时任务数量。**监控应部署在应用之外**，检测到连续心跳丢失时向真实值班团队告警，并按机构的人工备用救助流程处理。停机恢复后检查 outbox 重试状态、幂等键、通知回执与告警是否仍有人接手。

用户表中旧的 ADMIN 默认拥有主管权限，是迁移兼容策略；上线前在工作台逐一设置护理员、审计员、主管岗位。护理员与审计员已由统一岗位门禁限制为照护接口（仍按每个操作单独鉴权）和本人 MFA 设置，不能访问传统后台 ADMIN 接口；主管保留管理权限。**历史未分配岗位的 ADMIN 仍默认为主管**，上线前必须逐一清点与授权。这不是多机构、院区或老人级数据隔离。主管可检索最近敏感路径访问审计，但需另行制定保留期限、备份加密、历史审计归档和授权撤销制度。

部署前按 `docs/CARE_2.0.5_GATES.md` 对断网、服务中断、网关重复提交、设备 requestId 弱网重试、人员接单与二人复核进行真实硬件与人员演练。未通过此门槛不能把该源码预览版用于唯一的人身安全求助系统。
