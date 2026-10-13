# 变更记录：在原项目基础上做的改造

本文件说明当前 `智慧养老手表管理系统/` 相对最初那份原始项目（2019 网课模板）增加了什么。原始版本仍可通过 `git log` 与 tag `archive-2026-10-13` 找回。

## 一、设备侧（手表）接口

新增 `DeviceController`、`Device` / `DeviceDto` / `BindCode`、`DeviceJdbcDao`，接口前缀 `/restful/device`。

| 接口 | 用途 |
| --- | --- |
| `login` | 手表用 `deviceNo` + `secret` 换取会话 |
| `bind` | 手表用 6 位绑定码绑定老人（同步 `oldperson.device_id`、写入 `watch_info` 阈值） |
| `heartbeat` | 心跳，用于在线判断 |
| `report` | 上报心率 / 血压 / 步数 / 睡眠 / 经纬度 / 电量 / 摔倒告警 |
| `sos` | 一键求救 |

- 落库映射：心率血压 → `ratepre`（按阈值判定 `rate_state` / `pre_state`）；步数睡眠 → `watch_day_info`（每设备每天一条，`INSERT ... ON DUPLICATE KEY UPDATE`）；经纬度 → `device_location`；摔倒 → `alarm_precess(status=1)`；设备档案 → `device`。
- 参数范围校验：心率 20–250、血压 40–300、步数 ≤ 200000、睡眠 0–24 小时。
- 错误码：`0004` 鉴权失败、`0005` 未绑定、`0006` 绑定码无效。
- 后台管理接口：设备列表、新增设备、生成绑定码、重置密钥、解绑、删除、处理告警。

## 二、家属邀请制注册

- 新增 `InviteCodeController` / `InviteCode` / `InviteCodeDao`：管理员在「后台管理 → 邀请码管理」生成 8 位邀请码（字母表去掉易混的 `0 O 1 I`），可指定绑定老人、有效期与数量。
- 注册接口对匿名开放：校验邀请码 → 创建 `family` 记录 → 自动绑定该老人；一码一次，同一手机号只能注册一次。
- 前端新增 `Register.vue`（登录页「注册账号」入口，实时校验邀请码并回显绑定的老人）与 `inviteCode.vue`（生成 / 复制 / 作废 / 状态筛选）。

## 三、异常红点提醒

三类红点，处理后可标记，**同一件异常不会反复冒出来**：

| 红点 | 判定 | 处理方式 |
| --- | --- | --- |
| 人员不合格 | 未绑定 / 掉线 / 从未上报 / 电量 < 20% / 未配阈值 | 记录处理序号 `handled_seq`，只对"新的掉线"重新提醒 |
| 心率不合格 | 超出 `watch_info.hrf`–`hrl`（演示值 55–110） | 按"处理时间"比较，只统计处理时间之后的新记录 |
| 血压不合格 | 超出 `watch_info.bpf`–`bpl`（演示值 90–140） | 同上 |
| 摔倒 | `alarm_precess.status=1` | 在位置信息页处理 |

关键实现：`ratepre` 增加 `rate_handled_at` / `pre_handled_at`，`device` 增加 `person_handled_at` / `boot_seq` / `handled_seq` / `last_online_at`；掉线由 `scanOfflineDevices()` 判定（心跳超 120 秒未上报即计一次掉线）。

## 四、一套 Windows 统一入口

- `src-manager/Manager.cs` 编译出的 `智慧养老管理系统.exe`：菜单式启动 / 停止系统、初始化数据库、升级数据库（自动执行目录下所有 `*_migration*.sql`）、启动模拟手表、打开手册、查看账号、打开文件夹。
- 用 Windows 自带的 .NET 编译器即可重新生成：
  `C:\Windows\Microsoft.NET\Framework64\v4.0.30319\csc.exe /nologo /optimize+ /codepage:65001 /out:智慧养老管理系统.exe src-manager\Manager.cs`
- 原来的 8 个 `.bat` 入口功能都已被 EXE 覆盖：本地开发副本里它们移到了 `脚本备份/`，本仓库未收录（需要时可在 `archive-2026-10-13` 标签下取回），`脚本备份/` 只保留 4 份历史说明文档。

## 五、模拟手表

`watch-sim/watch-sim.js` 只依赖 Node 内置模块，Node 10 即可运行：

```sh
node watch-sim.js --server http://<电脑局域网IP>:8795 --deviceNo WATCH0001 --secret demo-secret-1 --interval 3 --scenario demo
```

`demo` 场景会循环演出「正常 → 心率偏高 → 正常 → 摔倒 → SOS」，用于演示；没有真机时就用它产生实时数据。数据库预置了 3 台演示设备：`WATCH0001` / `WATCH0002` / `WATCH0003`。

## 六、修掉的原有缺陷

| 问题 | 处理 |
| --- | --- |
| MySQL 8 默认 `ONLY_FULL_GROUP_BY` 让「位置信息」「心率血压」页面 500（旧 SQL 用非聚合列 GROUP BY） | 重写 `PositionJdbcDao` / `RatepreJdbcDao` / `StepJdbcDao` 的取数 SQL，不再依赖该 SQL 模式 |
| 心率血压、睡眠步数页面取不到最新记录（`GROUP BY` 随机取行，心率血压恒为 0） | 改为"先取每组最新时间、再回表取那一行" |
| 家属账号未关联老人时登录 500（`queryForObject` 抛 `EmptyResultDataAccessException`） | `findFamilyByPhone` 返回 `List` 取首条或 null，登录接口返回中文提示 |
| 删除老人后遗留未使用的邀请码 | 删除老人时一并清理 |

## 七、数据库脚本

- 全新安装：直接导入 `old.sql`（已含新表）。
- 老库升级：按顺序执行 `invite_code_migration.sql`、`device_migration.sql`、`alert_handled_migration.sql`、`alert_handled_at_migration.sql`（都可重复执行），或直接用 EXE 的菜单 `[5]`。
