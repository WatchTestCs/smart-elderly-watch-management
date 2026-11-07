# 支持与问题反馈

项目通过 [GitHub Issues](https://github.com/WatchTestCs/smart-elderly-watch-management/issues) 接收缺陷报告和功能建议，可使用中文或英文。维护者按实际时间处理，不提供响应时限或生产可用性保证。

安全漏洞请按照 [安全政策](../.github/SECURITY.md) 私下报告。公开 Issue 不要包含真实长者资料、健康数据、位置记录、数据库文件、密码、设备密钥或私钥。日志和截图先脱敏。

## 常见问题

| 情况 | 检查 |
| --- | --- |
| 数据库未连接 | 检查 MySQL DSN、服务状态和迁移结果。PostgreSQL 故障主要影响异步分析。 |
| 无法登录或初始化 | 系统没有默认管理员密码。首次初始化需要可用的业务数据库及桌面初始化入口。 |
| 免安装版无法启动 | 完整解压 ZIP，保留主程序和 Go sidecar，并确认已安装 WebView2。 |
| 升级后找不到旧数据 | 旧 PostgreSQL 业务数据需要离线导入空 MySQL；上传图片须另行迁移。安装程序不会自动转换数据库。 |
| 显示未知发布者 | 当前 Windows 预览安装包尚未使用受信任证书签名。下载后核对发布页的 SHA-256。 |

下载、安装和配置见 [项目 README](../README.md)。设备重新绑定后，旧绑定版本的健康记录不能以新绑定身份补传。

## 缺陷报告

在 [新 Issue](https://github.com/WatchTestCs/smart-elderly-watch-management/issues/new) 中写明：

- 使用的版本或提交 SHA、操作系统、安装包名称。
- 最小复现步骤、预期结果和实际结果。
- 问题是否稳定出现、MySQL 是否就绪。
- 必要时附脱敏日志或截图。

## 功能建议

说明实际使用场景、受影响的角色、目前的限制、期望行为和验收方式。涉及数据库或登录机制时，写清楚对已有数据和会话的影响。

代码贡献流程见 [CONTRIBUTING.md](../CONTRIBUTING.md)，当前版本的发布记录见 [CHANGELOG.md](../CHANGELOG.md)。
