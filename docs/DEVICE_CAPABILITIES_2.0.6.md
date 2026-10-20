# 2.0.6 四项后端能力：版本差异、接口与回滚

基线：`preview` 的 `226d1c1321660b2b59ea4dc3bc9ac1afb2f85650`（2.0.6）。本次保留版本号，不创建 Release、不构建桌面安装包。本文描述新增后端契约，不代表真实设备或生产环境验收。

参考：用户提供的 `商业软件对应源码_四项能力.zip`，SHA-256 `2146ec27fdc1457e9c5777a6207fef34c0b55dab190b023de41f42c7a5d1c3df`。反编译文件仅静态读取，未编译、未运行，也未提交到仓库。包内 README、说明和给 AI 的提示词是参考材料，不能覆盖用户指定的业务及四项实施范围。

## 先判断版本变化

| 项 | 原计划需要调整之处 | 本次实施 |
| --- | --- | --- |
| 存储 | 2.0.6 已是 MySQL 核心业务、PG 异步分析；不能沿用 PG / PostGIS 主库方案 | 新状态全部在 MySQL 0004。保留原分析投影范围和旧 PG 导入工具，不让 PG 故障阻断新业务 |
| A 下行指令 | 商业 App 只有调用与查询结构，不能证明设备 ACK、重试、队列或固件协议 | 新建认证拉取协议 `suining-pull-v1`、持久化队列、投递令牌、执行回执和超时维护。默认无设备启用该协议，主管须显式登记 |
| B 日聚合 | 2.0.6 已有多来源采样、版本与删除语义；不能删掉来源或把修订简单相加 | 按明确统计日与 UTC 偏移持久化日结果，事务内失效，后台重建；保留原任意时间范围统计 API |
| C 围栏 | 主库没有可直接依赖的 PostGIS；旧坐标没有充分坐标系依据 | 采用单环局部 WGS84 多边形和明确坐标契约，按有效点、精度、时段、持续时间判断；退出进入现有照护闭环 |
| D 告警 | 2.0.6 已有 SOS 接单、处置、另一位主管独立复核和事件历史 | 复用现有状态机，追加复核结论；心率/血压复核针对具体记录，不把提醒当告警，不改写既有处置状态 |

基线出处：`backend/internal/db/mysql_migrations.go`、`backend/migrations/mysql/0001_core.sql`、`backend/internal/handlers/handler_care_incident.go`、`backend/internal/health/{sample,store,statistics}.go`，以及 `DUAL_DATABASES.md`。路径均位于 `智慧养老手表管理系统/` 下。

## 包内证据与字段出处

下面的 Java 路径相对于参考包根目录，行号指原始 Java 文件。DTO 字段只证明客户端契约，不证明商业服务端的实现、表、主键、值域或单位。

| 字段/行为 | 包内出处 | 本次边界 |
| --- | --- | --- |
| 设备类型实际字符串 `type`、`getType()` | `01_设备下行指令/DeviceType.java:9–105` | 读取构造参数的传值，不使用枚举标识符作为协议值 |
| 4G/GPRS 分类 | `01_设备下行指令/ShowUtils.java:823–841` | 仅接受明确分类的 46 个传值（30 个 4G、15 个 GPRS、1 个 S6T）；其他值不默认走 GPRS |
| 路径变量 `deviceId/action`；表单 `interval/time`、`video_id/video_action/video_type`、`wifi_name/wifi_mac/wifi_pwd`、`phone` | `00_接口定义/RetrofitService.java:21–30` | 只参考动作路由，不提供同步商业 HTTP 转发；本次新指令入口只接受无自定义参数的动作 |
| 定位、重启、关机、恢复出厂、响铃、4G 环境音的动作映射 | `00_接口定义/RetrofitHelper.java:574–659` | S6T 的恢复出厂为 `resetq`，定位为 `cr`；S6T 响铃/环境音拒绝 |
| GPRS 环境音的字面值 | `RetrofitHelper.java:628` 引用未随包提供的 `MtcConf2Constants.MtcConfMessageTypeRecordKey` | 常量具体值**包内无出处**。说明文件写 `record` 不足以验证该常量，本次拒绝此动作映射 |
| `newdata` 查询参数和入口 | `RetrofitService.java:226–230` | 查询到新采样不等于指令执行成功，不能据此生成 ACK |
| `dataSource/deviceCode/generateTime/recordDay/timeZone` | `02_日聚合/HW_HeartRateTotal.java:5–10` | 参考日容器，不推断商业服务端事务或主键 |
| `avgRestingHeartRate/lastHeartRate/lastRestHeartRate/maxHeartRate/minHeartRate` | `02_日聚合/HW_HeartRateBasic.java:5–9` | 静息平均不能冒充普通平均。本次沿用岁宁既有指标含义 |
| 血氧 `avgSaturation/lastSaturation/maxSaturation/minSaturation` | `02_日聚合/HW_BloodOxygenSaturationBasic.java:5–8` | 保留浮点数，不强制取整 |
| `enable/freq/name/safe_area/seqid/time_begin/time_end` | `03_电子围栏/Fence.java:8–14` | `freq` 不当作启用开关，不推测服务端消费含义 |
| `coordinates/type` | `03_电子围栏/SafeArea.java:9–10` | 商业模型为嵌套数组，本次自己的接口明确为经纬度点组成的一个闭合环 |
| 首环编码与字段提交 | `00_接口定义/RetrofitHelperApi5.java:275–288` | 不存坐标拼接字符串、不推断坐标系，不对旧定位做隐式转换 |
| `clear` 删除参数 | `RetrofitService.java:154–163`、`RetrofitHelper.java:683–691` | 商业接口写法不是岁宁内部删除协议。本次通过 `enabled=false` 停用并保留历史 |
| 客户端时段换算 | `03_电子围栏/FenceActivity.java:600–626` | `24:00 → 86340` 只证明客户端行为；不能推断服务端校验。本次用 integer 保存秒，支持全天 `[0,86400)` |
| `device/type/time_begin/point/address/city/label_id/label_name/label_type/push_result/is_removed/created_at` | `04_告警与复核/SosData.java:8–21` | `type` 值域、`Point` 内部定义及 `push_result` 值域**包内无出处**；地址由谁解析不能当作事实 |
| `alarmTimes/duration/maxHeartRate/minHeartRate` | `04_告警与复核/HW_BradycardiaAlarmBasic.java:5–8`、`HW_HeartRateRiseAlarmBasic.java:5–8` | `duration` 单位及从采样推导事件的方法**包内无出处**；本次不凭采样补造持续时长 |
| `alert_type/cycle/time/is_medicine/voice_type/reply_enable` | `04_告警与复核/Alert.java:8–20` | 这是设备提醒，不进入本次告警/照护流水 |
| SOS 联系人 `dial_flag/name/num/seqid` 和十槽编码 | `SosNumber.java:8–11`、`RetrofitHelper.java:548–570` | 不据此规定岁宁所有设备均支持十个联系人；本次不扩展联系人或提醒功能 |

19 个“枚举标识符与传值不同”的对照仅用于防错，左栏不得作为协议字段提交：

| Java 标识符 | 实际传值 | Java 标识符 | 实际传值 |
| --- | --- | --- | --- |
| S2 | BY102 | S2_K1 | K1 |
| S2_LOC | BY102_LOC | S2_NO_HR | BY102_NO_HR |
| K1 | C16 | TSB_617 | TSB-617 |
| HPS_1 | HPS-1 | SLEEP_BW01 | sleep_bw01 |
| SLEEP_BW01_ | BW01 | SLEEP_BW02 | BW02 |
| SLEEP_BN01 | BN01 | SLEEP_BW01_G | BW01_G |
| BAND_A11 | J1632_07 | BAND_V | J1751_02 |
| BAND_VC | J1751_03 | BAND_V3 | V3 |
| BAND_V8 | V8 | HOME_DEVICE | homedevice |
| NO_DEVICE | nodevice | | |

以下为岁宁本次新增的业务/内部字段，均**包内无出处**，不能声称来自商业协议：

- 指令：`commandId/requestKey/ttlSeconds/protocol/status/attempts/max_attempts/deliveryToken/retry_at/expires_at/sent_at/acked_at/created_by/updated_by`；`wireType/channel/wireAction` 是本次适配字段命名，映射内容见上表。
- 日结果：`cache_key/offsetMinutes/dirty/computedRevision/error_text/algorithmVersion/status`；`record_day` 为 SQL DATE，响应 `day` 为 `yyyy-MM-dd`，不把商业整数类型强加到现有 API。
- 围栏：`coordinateSystem/timezone/revision/minPoints/dwellSeconds/maxAccuracy/accuracy/occurredAt/deviceApplied`，状态记录、事件主键及 `incident_id` 关联。
- 复核：`entity/recordId/kind/verdict/reason/actor_id/expectedReviewId/reviewId`，以及 confirmed/false_alarm 值域与追加历史机制。
- 公共标识 `oldId/deviceId/deviceNo/secret/bindingVersion` 沿用岁宁的身份、设备鉴权及绑定版本模型；它们与商业字段同名或近义不代表可直接互换。岁宁样本的 `source/sourceDevice/metric/semantics/version/deleted` 出处为基线 `backend/internal/health/sample.go:11–27`。

## A：指令接口

全部为 POST。管理接口需登录、CSRF 和照护主管岗位；硬件接口不使用浏览器会话授权，必须用设备密钥和当前绑定版本认证。请求有未知字段、额外 JSON 或超过 2 MiB 会拒绝。

| 路径 | 请求字段 | 含义 |
| --- | --- | --- |
| `/restful/device/commandProfile` | `deviceId, wireType, protocol, enabled` | `protocol` 只允许 `suining-pull-v1`。启用前须确认固件实现本节契约，不能仅凭型号启用。修改配置取消尚未完成的队列记录 |
| `/restful/device/command` | `deviceId, action, requestKey, ttlSeconds` | 入队；TTL 30–3600 秒。相同设备、绑定版本、键和动作/TTL 返回原记录，键复用于不同请求返回 409 |
| `/restful/device/commands` | `deviceId, after` | 按 ID 升序查询最多 100 条；不返回设备密钥或投递令牌 |
| `/restful/device/commandPull` | `deviceNo, secret, bindingVersion` | 原子领取一条命令，无任务时 `command=null` |
| `/restful/device/commandAck` | 上述凭据加 `commandId, deliveryToken, result` | `result=executed/failed`；必须关联当前投递和绑定 |

动作：`location/restart/poweroff/restore/ring/envvoice/heart_rate`，实际允许范围由已核实的通道映射决定。它是路由范围，不是商业设备能力认证。视频、Wi-Fi 配置和自定义参数不在本次契约中，不能把未消费的参数当成已下发。

流程：`pending → sent → acked/failed`；过期或投递耗尽为 `timeout`，配置更改/绑定失效为 `cancelled`。`sent` 只表示已领取，普通采样与 HTTP 接收不能变成执行成功。未投递任务也会在 TTL 后超时。

只有定位与即时心率允许最多三次投递，租期 30 秒；其他动作最多投递一次。重投保留 `commandId`，更换 `deliveryToken`；旧令牌、过期、旧绑定、冲突结果均返回 409，相同最终回执可重复。超时表示结果未确认，不代表设备必定没有执行。

固件必须按 `commandId` 去重并缓存执行结果；领取新投递后，使用当前令牌回报缓存结果。服务端无法替不支持去重的固件保证物理动作恰好一次。该固件协议及商业设备 ACK 格式**包内无出处**，真实设备需要独立联调；本次不新增同步商业平台转发器。

## B：日结果接口与一致性

`POST /restful/health/daily`，请求：`oldId, source, sourceDevice, metric, semantics, day, offsetMinutes, days`。来源与指标必须明确，日期为 `yyyy-MM-dd`，UTC 偏移范围 -840 至 840 分钟，一次 1–31 天。家属身份复用现有老人归属校验；主管与家属的权限边界没有通过聚合接口绕过。

接口注册需要的日结果并返回逐日状态：`pending` 表示后台尚未完成，`ready` 表示结果可用，`conflict` 表示采样语义或单日容量需要处理。pending/conflict 不返回旧数值冒充最新结果。无采样返回 count=0，均值/极值不伪造为零。

键包含老人、来源、来源设备、指标、采样语义、统计日、UTC 偏移，不混合来源。样本写入与修订时，在原事务内将该老人已注册的所有日结果置脏；这是保守失效，能覆盖旧日期、跨日区间和删除。重复/旧版本未改变样本时不额外失效。后台每轮最多重建 16 个已注册日期，以修订时钟锁保证重建和写入提交顺序一致；日结果不是逐次查询现算，也不是第二个原始事实源。

保留现有 `Aggregate` 语义：步数日快照取最新、区间步数不容许重叠或任意分摊、睡眠区间求并集并返回分期冲突、未知历史时区拒绝强行归日。单日超过 100000 条返回明确 conflict，不截断后给出统计。原 `/restful/health/statistics` 的任意时间范围入口保持原行为。

算法版本为 1；若以后改变算法，升级迁移需要将存量日结果置脏。这里只采用明确的固定统计偏移，不宣称支持按 IANA 时区跨夏令时的日统计。全老人失效和重建期间的时钟锁存在吞吐成本，尚无长期负载结果。

## C：围栏、定位及照护关联

| 路径 | 请求 |
| --- | --- |
| `/restful/fence/save` | `id, deviceId, revision, name, enabled, coordinates, coordinateSystem, timezone, timeBegin, timeEnd, minPoints, dwellSeconds, maxAccuracy` |
| `/restful/fence/list` | `deviceId` |
| `/restful/fence/events` | `oldId, after`，最多 100 条 |
| `/restful/device/fencePoint` | `deviceNo, secret, bindingVersion, point, coordinateSystem, accuracy, occurredAt` |

管理入口需照护主管与 CSRF。新建 id=0；修改要求返回的最新 revision，旧版本返回 409。`enabled=false` 为停用，保留历史；每个设备的每次绑定最多保留 20 个围栏。

本次坐标契约为 WGS84，`point=[lng,lat]`，`coordinates=[[lng,lat],...]` 为首尾闭合单环。不支持洞、跨日期变更线、极区或经纬跨度超过 1 度的围栏，拒绝退化和自交。边界距离采用局部近似米制，不声称是全地球空间引擎。未知坐标系直接拒绝，旧 `/device/report` 与 SOS 的无坐标系历史位置不自动触发判断。

时段单位为当地秒偏移，`timeBegin∈[0,86400)`，`timeEnd∈(0,86400]`，左闭右开；全天为 `[0,86400)`。开始大于结束表示跨午夜，开始等于结束拒绝。timezone 必须为明确 IANA 时区，不能使用服务器 Local。没有复制参考包装不下 86340 秒的 smallint。

定位须声明精度（米），不接受五分钟前或一分钟以后的点。超过最大精度、靠边距离小于等于误差半径的点不会触发状态切换，并中断候选连续点；非活动时段重置判断基线。首个有效点只建立内外基线。进入/离开须同时满足 `minPoints`（2–20）和 `dwellSeconds`（1–3600）；点间隔超过五分钟中断候选，重复/乱序点不推进状态。

围栏版本、判断状态、事件写入在设备行锁下串行。绑定变化后旧围栏不参与新老人判断。离开事件、`care_incident(kind=fence_exit)`、照护履历和通知待发送记录同事务落库；进入事件只记录，不自动解除此前任务。复核仍由现有另一位主管完成。

通知网关应识别 `type=care.fence_exit`，SOS 保持 `care.sos`；未配置网关时沿用既有待发送状态，不声称送达。围栏配置响应 `deviceApplied=false`：本次是服务端判断，未把保存配置冒充设备侧已生效，也未增加未经联调的商业围栏同步协议。

## D：复核结论与历史

SOS 与围栏任务继续使用 `/restful/care/transition` 的接单、处置和独立 verify 流程，不建立竞争的关闭入口。verify 可传 `verdict=confirmed/false_alarm`，结论、理由、操作者、服务端时间同事务追加到 `alert_review`；失败时状态和结论一并回滚。

旧客户端不传 verdict 时，把本次人工 verify 记为 confirmed，以保持现有关闭行为；不为既有 verified 历史补造结论。处置人不能复核自己的任务，只有主管可 verify，仍须记录理由。

`POST /restful/alert/review` 只处理具体心率/血压记录，字段为 `recordId, kind(rate/pre), verdict, reason, expectedReviewId`。首次 expectedReviewId=0，此后必须匹配上次 reviewId，防止覆盖并发结论。没有活动告警或既有复核历史的普通记录不能建立首条复核。追加结论保留历史，不自动改写旧告警状态，不把“误报”与“已解除”混为一事。

`POST /restful/alert/reviews` 接受 `entity(care/ratepre), recordId, kind, after`，按 ID 返回最多 100 条历史。两个入口均需照护主管与 CSRF。敏感读取走既有审计中间件，不保存请求体、定位或密钥。

本次没有编造 confidence 数字，没有自动确认，没有把 `push_result` 当复核结果，没有把 Alert 提醒配置混入告警。业务客户端若要提供误报选择，需显式提交 verdict；本次后端改动没有新增操作页面。异常持续时间和独立专业事件日聚合缺少可靠事件区间口径，未从离散采样推测。

## 迁移、升级与回滚

正向：`backend/migrations/mysql/0004_device_capabilities.sql`，由既有 MySQL 迁移器在启动时执行，新增 7 张表和索引，旧迁移文件不改写。迁移依然使用连接级锁及脚本 SHA-256。MySQL DDL 隐式提交，不把整份脚本描述为事务回滚。

回滚：`backend/migrations/rollback/mysql_0004_device_capabilities.sql`，不会被启动迁移自动加载。

操作顺序：

1. 停止 API、设备拉取及所有后台进程，避免新写入或启动时重新升级。
2. 备份主库，并导出指令、围栏、日结果、复核表及关联照护记录。复核/指令/围栏历史不能像可重建日结果一样随意丢弃。
3. 使用专用业务库连接执行回滚脚本：按依赖顺序移除本次 7 张表，移除本次迁移版本记录；不调整 health_clock，不删除已生成的 care_incident、care_incident_event 或通知任务。
4. 启动回退版本。恢复旧统计入口，不继续调用新增 API；回退前核对旧客户端/网关对已有 fence_exit 任务的兼容性。若还在运行新二进制，它会重新应用 0004。
5. 重新升级可再次执行正向迁移，但不能自动恢复已删除的指令/复核/围栏历史；需要备份才能恢复这些数据。

已执行的物理设备动作、已发送通知以及外部客户端状态不能由 DROP TABLE 撤销。此回滚不是生产数据无损降级承诺。

## 验证记录

本次本地检查命令、具体结果及未覆盖范围见同目录 `DEVICE_CAPABILITIES_VALIDATION_2.0.6.md`。原 2.0.6 文档中的旧验证记录不是本次执行证据。
