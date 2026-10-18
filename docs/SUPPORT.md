# Support and feedback / 支持与反馈

[简体中文 README](../README.md) · [English README](../README.en.md) · [Contributing](../CONTRIBUTING.md)

Use [GitHub Issues](https://github.com/WatchTestCs/smart-elderly-watch-management/issues) for reproducible non-sensitive bugs and feature requests. Chinese and English are welcome. Support is best effort; no response-time or production availability SLA is promised.

普通缺陷和功能建议可提交到 Issues。安全漏洞、真实人员资料、数据库文件、连接密码、设备密钥及私钥不要公开；请先阅读 [Security Policy](../.github/SECURITY.md)。

## Quick checks / 提交前排查

| Symptom | Check first |
| --- | --- |
| Database warning / 数据库未连接 | Confirm PostgreSQL is running and the configured address is reachable. Internet access alone does not identify a database. |
| Cannot create or use an account / 无法初始化或登录 | There is no default password. First-admin setup requires the native desktop flow and a ready database. |
| Portable app will not start / 免安装无法启动 | Extract the whole ZIP, keep both EXEs together, and confirm WebView2 is available; Setup can install the missing shared runtime. |
| Old records are missing / 升级后没有旧记录 | H2 is not converted automatically. Preserve old files and confirm the PostgreSQL migration plan; do not delete your only backup. |
| Linux differs from Windows / Linux 行为不一致 | Record the exact attachment name. Supplied Linux binaries are not established as builds of current Windows source. |
| Unknown publisher / 未知发布者 | Current Windows packages use local self-signing, not a publicly trusted CA. |

When sharing logs, include only a minimal redacted excerpt. Review screenshots for names, phone numbers, locations, and health information before posting them. Log collection is optional when it would reveal sensitive information.

## Bug report template / 缺陷报告模板

Copy this into a [new issue](https://github.com/WatchTestCs/smart-elderly-watch-management/issues/new):

```text
Summary / 问题概述:

Version or commit / 版本或提交:
Exact package filename / 安装包名称:
OS and architecture / 系统与架构:
Distribution / 交付方式: Windows Setup / portable ZIP / Linux archive / Linux DEB / source

Steps to reproduce / 最小复现步骤:
1.
2.

Expected / 预期行为:
Actual / 实际行为:

Database state / 数据库是否已连接:
Frequency / 是否稳定复现:
Redacted logs or screenshots / 脱敏证据（可选）:
```

Do not paste a connection string, production database, password, private key, or personal care record. A database status and redacted error are sufficient to start a report.

## Feature request template / 功能建议模板

```text
Use case / 使用场景:
Who is affected / 使用者:
Current limitation / 现有限制:
Desired outcome / 期望结果:
Alternatives considered / 已考虑的替代方法:
How to verify success / 如何验收:
```

Describe the user need before prescribing a framework or dependency. For database or authentication changes, explain compatibility and whether existing data could be affected.

## Branch and automation scope / 分支与自动化范围

Current community documents are maintained on `preview`; the default branch remains unchanged. These are copyable reporting templates, not a claim that GitHub's automatic issue-form chooser has been enabled. GitHub makes issue templates available after they reach the default branch; see [official template documentation](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository).

本轮只维护 preview 文件，不为启用自动表单而修改 main、默认分支或仓库设置。
