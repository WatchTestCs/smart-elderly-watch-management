# 位置与告警上下文增量契约

适用 /api/v1/device/report、/api/v1/device/sos。原凭据、绑定版本、请求幂等、经纬度和事件时间字段不变。

| 可选输入 | 含义及校验 |
| --- | --- |
| positionSource | 明确声明 gps、network、fused、manual、simulated 或 unknown；是设备声明，不是服务端传感器认证 |
| positionSampledAt | 带时区 RFC3339 时间，保存为 UTC；不得晚于服务端时间五分钟；允许历史采样，不用事件时间/接收时间补填 |
| positionAccuracy | 以米计的有限正数，必须为 JSON number；缺失不补默认精度 |

元数据只随本次完整经纬度提供；无坐标仍可 SOS，但不能携带不对应本次点的定位元数据。非法元数据返回 400，事务不落库。零经纬度是有效坐标，零精度不代表已知精度。

位置查询为最新定位返回 positionSource、positionSampledAt、positionAccuracy、positionValidity；告警独立返回 alarmSource、alarmSampledAt、alarmAccuracy、alarmValidity，以及 alarmKind。采样时间是 UTC RFC3339，精度是数字；未知来源、时间和精度为 NULL。beTime/alarmTime 继续表示原记录/事件时间，不冒充采样时刻。

有效性标记：reported 表示本次提供的坐标通过格式/范围校验，是未经独立核验的设备自报；missing 表示本次没有坐标；unknown 表示旧数据缺少可追溯上下文。reported 不是物理真实性、GPS 定位成功或精度可信证明。

明确 alarm=fall 的新普通告警为 fall，SOS 入口为 sos；只提供共用 alarm=1 的类型仍为 unknown，不从 status 或坐标猜测。旧行默认 unknown，不根据无法追溯的历史回填类型或来源。既有独立围栏来源不改；离线事件持久化类型仍属后续任务，不能宣布完整事件类型治理完成。

普通定位与告警各自保存元数据；新定位、重试 SOS、设备重绑定不会给旧告警补写来源、采样时间或精度。使用已有 location 字段加密机制封装带版本 JSON；无密钥的既有开发/测试模式与坐标行为一致，生产配置仍须通过原有密钥校验。

界面同时展示上报时间和采样上下文。采样距当前超过五分钟时标记历史位置；时间未知时明确不能判定为实时。该标记只影响呈现，不会丢弃历史点或抑制求助。

数据库：MySQL 0020_position_metadata.sql；PG 现有回归迁移 0018_position_metadata.sql。新增告警 kind 及两类快照 position_metadata，不删除/回填旧行。回退应用可保留新增列；MySQL 回滚脚本只撤销该迁移登记，重放保留已生成元数据。无生产发布或真实硬件/供应商验收结论。
