# P0 · 数据关系治理第一阶段：设备绑定引用完整性

此变更依赖 [PR #2（设备密钥 V3）](https://github.com/WatchTestCs/smart-elderly-watch-management/pull/2)，它又依赖 [PR #1（10 月 15 日遗留修复）](https://github.com/WatchTestCs/smart-elderly-watch-management/pull/1)。**不得跳过祖先 PR 直接合并**。

## 当前模型（从源代码核验）

- `oldperson.old_id` 是长者 ID；`device.old_id` 存放当前手表绑定的长者 ID，属于**可空的活动绑定引用**。
- `oldperson.device_id` 是历史前端兼容用的反向设备显示引用，可能存在**过期、重复或缺失**；本阶段不将它定义为关系权威来源，也不自动清洗。
- `watch_info.device_id` 用于手表健康阈值；`ratepre`、`device_location`、`watch_day_info`、`alarm_precess` 是历史事实记录，不能因解除当前绑定而级联删除。
- `family_id`、`customer_id` 的语义和实际来源还需后续独立审查；不可简单补外键并删除孤儿记录。

## 本阶段 V4 改造及明确约束

1. Flyway Java V4 在创建外键前检测所有 `device.old_id` 孤儿值；如引用缺失的 `oldperson.old_id`，迁移直接失败、拒绝启动，不静默更新/删除任何记录。文件数据库已有 `DatabaseMigrations` 的**迁移前备份**。
2. 创建 `device.old_id -> oldperson.old_id` 外键，`ON DELETE RESTRICT`；允许未绑定设备的 `NULL`。**不做** `ON DELETE CASCADE` 或自动 `SET NULL`，避免清除关联和影响告警追溯。
3. 管理员删除长者前检查有效绑定；存在设备时返回 HTTP 409，并且在删除邀请码之前拒绝，要求先明确解绑。DAO 的数据库外键是竞争情况下的最终保护；事务确保异常不会消费邀请码。
4. **不变式**：任何新设备绑定必须指向现存长者；删除被绑定长者不能孤立设备；解除绑定不得删除历史测量/告警事件；升级错误不自动修改历史数据。

## 验收（CI 与外部证据分离）

- 新库 V1→V4、二次启动、原有版本前进/回退拒绝测试。
- V3 文件库正常升级至 V4；缺失长者的孤儿设备引用导致失败，且 H2 升级前 ZIP 可恢复、引用和 `schema_version=3` 未被静默篡改。
- 创建非法引用、更新非法引用、删除被绑定长者全部由数据库拦截；解除绑定后可以删除。
- 控制器在活动绑定存在时返回 409 且邀请码完整；原有取消/失败事务测试持续通过。
- 真实用户数据升级仍需在**离线脱敏备份副本**上演练，不能把夹具测试当成生产数据验收。

## 后续必须独立设计

- 一致性矩阵：`device.old_id` vs `oldperson.device_id`，包括多设备绑定一个长者、换绑、解绑后历史记录的显示。禁止一键覆盖反向 ID。
- 主体关系：长者-家属/监护人/管理员的权限映射，以及软删除、审计追溯和数据导出。
- 历史事实数据与活动绑定解耦。后续引入 `device_binding` 生命周期表前先确定解绑、继承历史、时间范围与隐私策略；必要时迁移为 append-only binding history，并通过安全回填证明。
- 其他表的外键/索引必须逐表在已有真实数据分布上验证；不可一次性补齐所有关系。

如需回滚：停止应用，确认备份路径、完整性和版本，恢复 V4 升级前 ZIP，再用 V3 版本运行。禁止试图通过删除 `flyway_schema_history` 或手工把 `app_metadata` 改回旧版本降级。
