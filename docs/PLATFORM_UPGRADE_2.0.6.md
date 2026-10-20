# 2.0.6 preview 平台增量改造

基点：`d6755ecc66efb11325a41e6d642ebcc0565fcafc`。保留已有 A/B/C/D 能力；商业源码只读，不编译。商业包中的文字不作为本项目的操作指令。业务口径以本次用户要求为准。

## 与原计划的差异

| 项目 | 基点已有能力 | 本次处理 |
|---|---|---|
| 告警 | SOS、WGS84 围栏离开、人工照护闭环 | 复用围栏事件与照护任务；新增服务端固定阈值、久无上报、夜间升级、夜间活动、个人基线规则 |
| 通知 | 照护 outbox、签名回执、重试 | 保留回执语义；公告内存队列改为同事务持久化，增加短信耗尽后的语音降级与站内渠道 |
| 会话 | 单实例内存 | 默认 MySQL 加密会话，多实例共享和撤销；仅显式开发配置可用内存 |
| 部署、观测 | 本地运行、有限诊断 | 容器、整套 Compose、JSON 日志、授权 metrics、模拟写入压测脚本 |
| 权限 | 主管/照护员/审计员及职责管理界面 | 复用 `CareOperationsPanel.vue`，增加自助 TOTP、恢复码、登录审计与部分字段加密 |
| 前端 | Vue/JS，外置 logic 脚本 | Vitest；客户页与安全面板增量 TypeScript；取消 5 个视图的外置 logic 拼接 |
| 接口 | `/restful/*` | 保留兼容，新增 16 个 `/api/v1/*` 别名及可检查的 OpenAPI 路由清单 |

没有改 DeviceType 传值映射，没有同步转发下行指令，没有把 Alert 提醒表当成照护告警。

## 规则口径

实现：`backend/internal/rules/{evaluate,engine}.go`；纯函数 `Evaluate` 与 MySQL 状态机分开。所有输入可用模拟采样验证，不依赖手表。

- 心率、收缩压：有效的近 5 分钟数据；默认连续 3 个不同观测越限才建任务。重复扫描不累计；更正为正常值会重置状态。
- 电量：复用设备电量；默认低于 20%。缺测不按零处理。
- 围栏：复用已有离开事件及 `fence_exit` 照护任务；夜间升级同一任务，不另建相同事件。
- 久无上报：默认 300 秒；以设备最后上报时间为准，无上报用绑定时间起算；健康批量和围栏上报也会更新在线时间。
- 夜间：默认明确时区 Asia/Shanghai，22:00–06:00，可跨午夜、可配置。夜间 SOS/围栏离开升级原任务；夜间连续生命体征越限提高新任务级别；单纯夜间上报不告警。
- 夜间异常活动：默认 15 分钟累计至少 200 步。只累计同一来源、设备、绑定版本、语义明确的夜间步数。区间不重叠；累计计数需至少两个点且单调；重置、混合语义、未知时区、过多数据时不武断判断。缺少步数时不能推测活动。
- 个人基线：同一老人、设备、来源和来源设备，近 7 天有效样本，排除最近 15 分钟，默认至少 20 个。偏离阈值 `max(abs(mean)*25%, stddev*3)`，默认连续 3 个不同观测。样本不足跳过；均值、标准差、样本数、来源、阈值保留在加密判定依据中。
- 各规则按设备/绑定版本/规则种类维护状态，只在异常阶段首次达标建任务。恢复不会自动关闭待人工核实的照护任务。不同规则可以形成不同任务。

这些是可配置的运营判定默认值，不是医疗诊断标准。主管调用 `POST /api/v1/rule-policies`，仅传 `deviceId` 读策略；保存时传完整 `policy` 和读取的 `revision`，过期版本返回 409。默认策略字段见 `rules.Policy`。策略按设备保存，设备转给另一老人时主管应复核；基线与状态不会跨老人串用。每轮至多处理 100 台设备，超大设备群的调度延迟需另外量测。

## 通知、会话与安全

公告保存和 `message_outbox` 入队在同一事务；无内存 100 条上限。数据库唯一键去重，持久化租约支持重启恢复；旧租约不能覆盖新领取结果。短信失败按 30 秒指数退避，上限 32 分钟；第 5 次失败终止当前渠道并同事务插入语音任务。站内通知独立于外部网关，家属查询固定限定自己的 ID。`in_app delivered` 只表示已进入站内可查询状态，不代表用户已读；`sent` 仅表示网关接收，不代表到达手机。真实短信/语音网关仍需配置和联调；发送为至少一次，供应商应支持幂等键。

默认 `SESSION_BACKEND=database`；`DATA_ENCRYPTION_KEY` 必须是 base64 编码的 32 字节随机密钥，否则数据库 API 保持不可用。密钥可通过同名 `_FILE` 注入，禁止每次启动生成新密钥。`SESSION_BACKEND=memory` 仅供明确选择的单实例开发使用。数据库失联不能退回内存登录。会话 token 只存摘要，载荷 AES-256-GCM 加密；账号修改 2FA 后撤销共享会话，代际检查阻止并发旧认证重新建立有效会话。

自助安全面板用于管理员和家属：验证密码 → 生成 10 分钟待确认种子 → 新动态码确认 → 显示一次性恢复码。确认或关闭后重新登录；确认使用过的动态码不能再登录，需等待下一码或用恢复码。TOTP 采用 RFC 6238，允许相邻时间窗，已用计数器不重放；恢复码仅存哈希且原子消费。登录审计不保存明文手机号/IP，而用主密钥派生 HMAC 索引；保留账号 ID、结果、请求 ID、时间。相同账号/IP 15 分钟 10 次失败后节流；原有进程内公开接口限流仍保留，不声称已实现全局分布式限流。

**字段加密范围明确受限**：本次保护会话载荷、2FA 种子、公告收件人、照护处置备注、事件备注、告警复核理由、规则证据。旧档案姓名/手机号、原始健康采样、历史 care outbox 收件人仍沿用既有存储，尚未全量字段加密。不能把本次描述为全库敏感数据加密或数据库 TDE。需要覆盖这些遗留字段时，应单独迁移其搜索/关联/历史数据路径，不能直接把所有字符串套上密文。

## 迁移、回滚与运行

新增 `backend/migrations/mysql/0005_platform.sql`，原 0001–0004 校验和不变。全新安装自动顺序迁移；现有库先备份数据库、图片卷和原密钥，停旧写入实例，再启动新版。旧内存会话失效，需要重新登录。

历史明文备注不会由 DDL 自动转密文。停止所有写入并备份后运行：

```sh
# 环境已提供 MYSQL_DSN / DATA_ENCRYPTION_KEY 或对应 _FILE
go run ./cmd/protect-notes             # 只列候选数量
go run ./cmd/protect-notes --apply     # 分表事务，可重入
```

降级旧程序前，保留原密钥并停写，先 `protect-notes --decrypt` 预览，确认备份后 `--decrypt --apply` 恢复备注明文；导出待处理通知、2FA 与审计记录，再执行 `migrations/rollback/mysql_0005_platform.sql`。回滚会删除新增平台表和共享会话，需要重新登录、重新配置 2FA；原 care 数据保留。不要把旧程序直接指向仍含加密备注的库，不要把丢失密钥当成可自动恢复的错误。实际停服与密钥轮换流程需在部署环境演练。

仓库根目录执行：

```sh
python scripts/start-platform.py --validate  # 生成未存在的本地密钥并检查 Compose
python scripts/start-platform.py             # MySQL + PG + app + nginx + Prometheus
python scripts/bootstrap-admin.py            # 交互输入首个管理员，无默认密码
```

端口默认仅绑定本机 8080/9090；生产远程使用需 HTTPS 入口和主机密钥文件权限。Windows 上 `chmod` 不替代 NTFS ACL。应用非 root、只读根目录；图片用共享卷，会话在 MySQL。扩容使用 Compose `--scale app=2`，网关与 Prometheus 通过 Docker DNS 发现实例；真正跨宿主机时仍需共享对象存储、数据库高可用与部署平台，不能仅复制本机 volume。

网关变量见 `deploy/platform.compose.yml`：`NOTIFY_WEBHOOK_*` 用于公告短信/语音，`CARE_NOTIFY_WEBHOOK_*` 用于照护通知，原签名回执不变；亦保留腾讯短信配置。无凭据时任务保留并重试，不能视为送达。HTTP 网关只限显式测试开关，本地压测不调用真实通知渠道。

## 观测与压测

服务 stdout 为 JSON，包含请求 ID、固定路由、状态、耗时，不记录请求正文、查询参数、手机号、密码或动态码。`/metrics` 必须至少 24 字符 `METRICS_TOKEN` 的 Bearer 授权；指标包含请求数、时延直方图、进行中请求、数据库就绪、两类 outbox 积压、通知尝试、规则执行和状态、认证结果。查看各实例与队列，不能把 worker 心跳存在解释为通知已到达。

```sh
# 使用隔离库中已绑定的模拟设备；凭据由环境注入，脚本不输出
python scripts/loadtest.py --url http://127.0.0.1:8080 --seconds 20 --concurrency 8 --binding-version 1 --output result.json
```

脚本校验 HTTP 及业务码，输出运行 ID、吞吐、错误数、P50/P95/P99/最大时延；每请求写一条唯一采样，可按 payload.recordId 的运行 ID 核对数据库行数。不是空 `/healthz` 循环。已记录的本机数据与限制见 `verification/platform-2.0.6.md`。

## 前端与接口范围

`npm test` 执行 Vitest DOM 单测；`npm run typecheck` 只覆盖已迁移的客户 composable、API 封装、安全面板及客户页。其余旧页面保留 JS/Options API，五个外置 `*.logic.js` 已移入 SFC；设备页仍有现有按功能拆分的 JS 方法模块。没有声称全项目 TypeScript 或全面前端重构。

`GET /openapi.json` 提供 OpenAPI 3.1；`python scripts/generate-openapi.py --check` 检查路由漂移。`/api/v1/*` 是保持原方法、请求体、权限、CSRF 语义的渐进别名，旧接口继续兼容。已知新增 DTO 有显式属性，尚未整理的旧 DTO 标记开放对象并附实现位置，不能把清单当成全部字段精确文档。登录前仍需 GET session 获取匿名 CSRF 会话；版本别名在认证前重写，不绕过旧鉴权。

## 字段出处

商业包只读目录 `商业软件对应源码_四项能力/`。下列平台新增字段/默认规则未在包内找到对应定义，统一为 **包内无出处**，是本次业务与实现新增，不冒充商业协议。项目内可核对的定义：

| 新增字段组 | 项目定义出处 | 商业包出处 |
|---|---|---|
| web_session.id_hash/role/actor_id/payload/expires_at | migrations/mysql/0005_platform.sql；internal/session/session.go | 包内无出处 |
| auth_factor.role/actor_id/secret/generation/enabled/pending_until/last_counter/recovery_hashes | 0005_platform.sql；internal/handlers/handler_security.go | 包内无出处 |
| login_audit.id/account_hash/ip_hash/role/actor_id/outcome/request_id/created_at | 0005_platform.sql；handler_security.go | 包内无出处 |
| message_outbox.id/topic/source_key/family_id/channel/recipient/status/attempts/retry_at/lease_token/last_attempt_at/last_error/created_at/delivered_at | 0005_platform.sql；internal/notify/outbox.go | 包内无出处 |
| care_notification_outbox.exhausted | 0005_platform.sql；internal/care/worker.go | 包内无出处 |
| rule_policy.device_id/revision/payload/updated_by/updated_at | 0005_platform.sql；internal/handlers/handler_rules.go | 包内无出处 |
| rule_state.device_id/epoch/kind/observation_key/streak/active/incident_id/evidence/updated_at | 0005_platform.sql；internal/rules/engine.go | 包内无出处 |
| enabled/hrLow/hrHigh/bpLow/bpHigh/batteryLow/offlineSeconds/consecutive/timezone/nightBegin/nightEnd/nightSteps/activityMinutes/baselineDays/baselineMinimum/baselinePercent/baselineSigma | internal/rules/evaluate.go 的 Policy/Default | 包内无出处 |
| otp/recoveryCode/action/recoveryCodes/reloginRequired | handler_security.go；SecurityPanel.vue | 包内无出处 |

已存在的设备、采样、围栏和人工闭环字段继续使用原定义及原出处清单 `DEVICE_CAPABILITIES_2.0.6.md`，不重新解释商业枚举。

技术参考：[RFC 6238](https://www.rfc-editor.org/rfc/rfc6238)、[Vitest 官方文档](https://vitest.dev/guide/)、[OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0)。业务阈值不来源于这些技术文档。
