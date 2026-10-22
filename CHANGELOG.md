# Changelog / 更新日志

This is a concise user-facing history, not a complete commit log. Release notes and artifact manifests retain version-specific validation details. / 本文件记录主要变化，完整验收与附件来源以相应 Release 为准。

## 2.1.0 — Windows 完整交付与后端改进

[发布与升级说明](docs/releases/2.1.0.md)

- 同一 preview 提交生成 Windows x64 安装、免安装和完整源码包，产品版本统一为 2.1.0，附 SHA-256 和来源清单。
- 整合 2.0.6 之后的 Go 1.23 适配与后端修复，以及 PR #24 的持续排空、队列诊断采样和日缓存写入优化。
- PR #25 默认保留 20 个 MySQL 空闲连接，增加受限配置与回退方式；微基准收益不外推为 HTTP 端到端性能。
- 打包流程增加隔离目录中的安装、免安装启动及卸载保留资料检查；真实硬件与生产验收边界保留。

## 2.0.6 — UI、照护闭环与双库整合

[改进与升级说明](docs/releases/2.0.6.md)

- 统一导航与标题，按当前路由折叠菜单，补齐窄屏卡片、加载与失败重试，优化总览宽屏布局。
- 修正建档/设备绑定依赖、归属一致性、阈值保留、饮食药物字段错位及缺测虚假正常。
- 打通 SOS 接单、处置、独立复核、关联告警关闭、总览刷新和分页履历。
- MySQL 承担业务事务，PG 接收可靠异步分析；附双库部署、旧库导入及投影回填工具。
- 保留远程健康采样、绑定版本、增量游标与统计，适配 MySQL；离线导入兼容 PG 0006／0007／0008。验证与安装包边界见 [整合记录](docs/INTEGRATION_2.0.6.md)。
- 统一应用、后端、前端及锁文件产品版本为 2.0.6。本文记录本地交付，未创建远程 Release 或移动标签。

## Unreleased — documentation on preview

- Add an English README with language navigation and the same platform, database, and signing boundaries as the Chinese overview.
- Add contribution guidance, copyable issue/feature templates, support notes, and release-history navigation.
- Use static license and preview-version badges; do not imply successful CI, audit certification, or a support SLA.
- Documentation-only: no binary rebuild, new application version, or movement of existing release tags.

## 2.0.2 — Tauri / Go and independent PostgreSQL

[Release: v2.0.2-preview.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.2-preview.1)

- Replace the Windows Electron/Java stack with Tauri 2 / Rust and a Go sidecar, retaining Vue 3 / Vite.
- Keep PostgreSQL independently deployed. Open the UI without a database; show local reminders, reconnect in the background, preserve active input, and never replay failed writes automatically.
- Keep authenticated logout available without PostgreSQL while retaining session and CSRF checks.
- Replace the originally published Windows bundle with a lightweight Setup at the user's explicit request, retaining version 2.0.2. Reuse system WebView2 and silently install it only when missing.
- Add Windows portable and exact-commit source ZIPs. Add user-supplied Linux application and database-manager attachments with explicit provenance and runtime-validation limits.
- Legacy H2 data is not converted automatically. Windows binaries are locally self-signed, without public CA trust.

## 2.0.1 — historical Windows preview

[Release: v2.0.1-preview.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.1-preview.1) · [Archived notes](docs/releases/2.0.1.md)

- Strengthen device-secret storage, relational integrity, deletion handling, and regression coverage.
- This release belongs to the earlier Java desktop architecture; its build and validation claims must not be reused as evidence for the current Tauri / Go implementation.

## 2.0.0 — historical Windows preview and later Linux attachment

[Release: v2.0.0-preview.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.0-preview.1)

- Historical Windows delivery used Electron, Java, and H2, with the Vue 3 / Vite frontend.
- A user-provided Linux desktop DEB was later attached unchanged. Maintainer familiarity is limited; it may contain bugs, errors, or compatibility issues and has not undergone installation or runtime validation.

Older implementation descriptions are preserved as history and are not current setup instructions. / 旧架构仅作历史，不作为当前部署依据。
