# 独立部署照护后台（2.0.4）

此部署方式使 Go 后端与桌面 GUI 的退出独立。**单实例并不等于真正高可用**：硬件接入、短信运营商、数据库、网络、供电和现场救援均需要另行保障。

## 运行模式

- 在 Linux 服务器持续运行 `backend/cmd/server` 编译的 Go 程序，Windows 桌面端只作为客户端使用，不启动/终止这个服务端进程。
- PostgreSQL 需持久化、定期备份并演练恢复；所有数据库迁移在启动前由服务执行，首次升级前必须备份。
- `/healthz` 仅表示 HTTP 服务可连接；`GET /restful/service/status` 的 `databaseReady` 字段用于 DB 就绪诊断。注意 Web 和通知网关可用性要另行监控，不能把 HTTP 200 当作 SOS 可被救援的证明。

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
DATABASE_URL=postgres://suining:REPLACE@127.0.0.1:5432/suining?sslmode=require
IMAGE_DIR=/var/lib/suining/images
CARE_NOTIFY_WEBHOOK_URL=https://your-trusted-gateway.example/api/notify
CARE_NOTIFY_WEBHOOK_TOKEN=REPLACE_WITH_SECRET
CARE_RECEIPT_SECRET=REPLACE_WITH_RANDOM_SECRET_AT_LEAST_32_CHARACTERS
CARE_ESCALATION_PHONE=REPLACE_WITH_ON_CALL_PHONE
```

API 不应直接暴露到公网，建议用可信 TLS 反向代理和网络 ACL。以上 `ADDR` 地址与数据库连接字符串须按实际部署更改；若服务跨机器供设备接入，监听地址及公网域名/防火墙等也须独立配置。

## 通知状态与网关契约

SOS 成功响应是 `received`，不是 `delivered` 或 `acknowledged`。若有已绑定家属手机号，将在同一事务生成状态 `pending` 的 outbox 记录。未配置网关时任务安全留在数据库等待部署人员处理。后台每约五秒领取待发送任务，带有 `Idempotency-Key: care-outbox-<notificationId>` 的 POST JSON 请求；**网关必须持久化该键并对重复发送去重**。HTTPS 2xx 表示网关已接收（`sent`），不是已送达。

网关确认真实送达后必须带以下请求调用 `POST /restful/care/receipt`：
- JSON：`{"notificationId":123,"status":"delivered"}`
- `X-Care-Timestamp`：当前 Unix 秒
- `X-Care-Signature`：HMAC-SHA256 十六进制，密钥 `CARE_RECEIPT_SECRET`，消息为 `timestamp + "." + 原始 JSON 字节`

该接口拒绝无签名、过期签名、未处于 `sent` 的任务或重复回执。生产网关需要独立端到端验收。仅靠供应商 HTTP 2xx 不应填写“家属已收到”。

## 人工处置和升级

管理员使用 `/restful/care/incidents` 查询、`/restful/care/transition` 接单/开始处置/登记措施/复核；每个动作均形成审计事件。超过当前时限而未完成时，服务将写入 `escalated` 并对 `CARE_ESCALATION_PHONE` 创建额外的待发通知（如已配置）。超时升级是软件流程，仍不等于救援人员已到场。

应对未完成任务设置独立可观测性：定时统计超过期限、未接单、网关发送失败和服务下线的任务，由值班团队在外部渠道进行确认。对生命安全事件，应保留电话/人工备用求助方式。

## 运维与恢复检查

至少每日完成备份，并在隔离环境定期演练 `pg_restore` 和上传图片数据的恢复；留存校验和、恢复时间目标和备份删除策略。监控数据库连接、网关退避队列长度、最长未接单时间、告警升级数量及服务退出次数。

当前仓库不能证明真实硬件、短信运营商、应急人工值守和多机故障转移通过；这些都是正式交付的门槛。
