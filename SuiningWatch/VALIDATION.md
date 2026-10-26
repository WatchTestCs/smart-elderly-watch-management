# 原独立工程历史验证记录

以下记录属于 2026-10-25 的原独立目录，不是当前 Git SHA 的重新验收。本次导入的协议测试、
原生构建与后端联调单独记录于 ../docs/WATCH_INTEGRATION.md 及对应证据，历史 artifacts 不入库。

验证日期：2026-10-25。工程为 ArkTS/ArkUI Stage Wearable 应用，bundleName 为 `com.suining.watch`。

| 检查 | 结果与边界 |
| --- | --- |
| API 24 SDK 原生编译 | 成功，生成未签名模拟器 HAP；编译器仍有 SDK 可能抛异常的静态警告，调用入口通过 try/catch 处理 |
| 原生协议逻辑测试 | 6 项通过，直接编译加载生产 Protocol.ets |
| 隔离 Go + MySQL 协议联调 | 6 项检查通过，见 artifacts/e2e/result.json |
| 采样去重 | 同一批重传返回 2 条重复记录，健康变化记录仍为 2 条 |
| 绑定隔离 | 解绑重绑后 epoch 变化，客户端拒绝旧队列，后端对旧版本请求返回 409 |
| SOS 幂等 | 同一 requestId 返回同一 incidentId；测试无实际通知接收人 |
| Wearable 模拟器安装 | HDC 报 install bundle successfully；API 24、466×466、320 dpi |
| Ability 启动 | HDC aa start 成功，能力管理记录显示应用为 FOREGROUND；运行日志确认 `SuiningWatch: Index page loaded` |
| 界面视觉和完整点击 | 未验证：本次 DevEco 与模拟器的 Computer Use 界面访问未获授权 |
| 真机安装与传感器 | 未验证，不应将模拟数据或模拟器安装称为真实手表健康采集 |

不修改网页与后端源码。新代码全部位于独立 SuiningWatch 文件夹。构建复用本机 IDE 工具，临时链接和构建缓存已在 .gitignore 中排除。
