# 2.0.2 文件职责索引

列出当前提交中的全部源码、配置、文档和视觉资源。构建缓存、用户数据及 Release 安装包不进入源码仓库。GitHub 文件列表显示最近一次提交消息，逐文件职责以本索引为准。

| 文件 | 职责 |
| --- | --- |
| [.github/CODEOWNERS](../.github/CODEOWNERS) | 项目文件 .github/CODEOWNERS 的实现与配置（.github/CODEOWNERS）。 |
| [.github/SECURITY.md](../.github/SECURITY.md) | 项目文件 .github/SECURITY.md 的实现与配置（.github/SECURITY.md）。 |
| [.github/dependabot.yml](../.github/dependabot.yml) | 项目文件 .github/dependabot.yml 的实现与配置（.github/dependabot.yml）。 |
| [.github/pull_request_template.md](../.github/pull_request_template.md) | 项目文件 .github/pull_request_template.md 的实现与配置（.github/pull_request_template.md）。 |
| [.github/workflows/ci.yml](../.github/workflows/ci.yml) | GitHub 工作流 Preview CI 的触发条件与执行步骤（.github/workflows/ci.yml）。 |
| [.github/workflows/desktop-contract.yml](../.github/workflows/desktop-contract.yml) | GitHub 工作流 Desktop Contract 的触发条件与执行步骤（.github/workflows/desktop-contract.yml）。 |
| [.github/workflows/p0-db-migration.yml](../.github/workflows/p0-db-migration.yml) | GitHub 工作流 PostgreSQL migration verification 的触发条件与执行步骤（.github/workflows/p0-db-migration.yml）。 |
| [.github/workflows/preview-candidate.yml](../.github/workflows/preview-candidate.yml) | GitHub 工作流 Tauri preview candidate 的触发条件与执行步骤（.github/workflows/preview-candidate.yml）。 |
| [.github/workflows/preview-checks.yml](../.github/workflows/preview-checks.yml) | GitHub 工作流 Additional preview UI checks 的触发条件与执行步骤（.github/workflows/preview-checks.yml）。 |
| [.github/workflows/preview-release.yml](../.github/workflows/preview-release.yml) | GitHub 工作流 Preview release preparation 的触发条件与执行步骤（.github/workflows/preview-release.yml）。 |
| [.github/workflows/reusable-backend.yml](../.github/workflows/reusable-backend.yml) | GitHub 工作流 Reusable backend verification 的触发条件与执行步骤（.github/workflows/reusable-backend.yml）。 |
| [.github/workflows/reusable-frontend.yml](../.github/workflows/reusable-frontend.yml) | GitHub 工作流 Reusable frontend verification 的触发条件与执行步骤（.github/workflows/reusable-frontend.yml）。 |
| [.github/workflows/reusable-windows-package.yml](../.github/workflows/reusable-windows-package.yml) | GitHub 工作流 Reusable Tauri contract 的触发条件与执行步骤（.github/workflows/reusable-windows-package.yml）。 |
| [.github/workflows/scorecards.yml](../.github/workflows/scorecards.yml) | GitHub 工作流 OpenSSF Scorecard 的触发条件与执行步骤（.github/workflows/scorecards.yml）。 |
| [.github/workflows/security.yml](../.github/workflows/security.yml) | GitHub 工作流 Security 的触发条件与执行步骤（.github/workflows/security.yml）。 |
| [.github/workflows/ui-quality.yml](../.github/workflows/ui-quality.yml) | GitHub 工作流 UI Quality 的触发条件与执行步骤（.github/workflows/ui-quality.yml）。 |
| [.github/workflows/windows-build-smoke.yml](../.github/workflows/windows-build-smoke.yml) | GitHub 工作流 Tauri Windows build 的触发条件与执行步骤（.github/workflows/windows-build-smoke.yml）。 |
| [.gitignore](../.gitignore) | 仓库级排除规则，防止依赖缓存、生成包、用户数据库与签名私钥进入 Git。（.gitignore）。 |
| [CHANGELOG.md](../CHANGELOG.md) | 面向使用者的版本历史，区分当前架构、历史发布和未发布文档（CHANGELOG.md）。 |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | 中英贡献指南：preview 目标、聚焦修改、按风险验证与安全协作（CONTRIBUTING.md）。 |
| [LICENSE](../LICENSE) | 仓库整体 MIT 许可与 WatchTestCs 版权声明。（LICENSE）。 |
| [README.en.md](../README.en.md) | 英文项目首页：功能、下载、独立数据库、开发入口与验证边界（README.en.md）。 |
| [README.md](../README.md) | 岁宁 · 智慧养老管理：对应主题的当前使用、实现或验证说明（README.md）。 |
| [architecture.md](../architecture.md) | 岁宁架构总览：对应主题的当前使用、实现或验证说明（architecture.md）。 |
| [docs/ABOUT.md](../docs/ABOUT.md) | 仓库 About 文案：对应主题的当前使用、实现或验证说明（docs/ABOUT.md）。 |
| [docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md) | 实现架构 · Tauri / Go / Vue / PostgreSQL：对应主题的当前使用、实现或验证说明（docs/ARCHITECTURE.md）。 |
| [docs/BACKEND_SCREEN_REFINEMENT.md](../docs/BACKEND_SCREEN_REFINEMENT.md) | 后端与屏幕布局优化验收 · 2026-10-16：保留的历史决定或验收记录，请以 2.0.2 当前指南为准（docs/BACKEND_SCREEN_REFINEMENT.md）。 |
| [docs/BUILD.md](../docs/BUILD.md) | Windows 构建 · 2.0.2：对应主题的当前使用、实现或验证说明（docs/BUILD.md）。 |
| [docs/CONTROL_REFINEMENT.md](../docs/CONTROL_REFINEMENT.md) | 控件边界与工具栏验收记录：保留的历史决定或验收记录，请以 2.0.2 当前指南为准（docs/CONTROL_REFINEMENT.md）。 |
| [docs/DATABASE_AVAILABILITY.md](../docs/DATABASE_AVAILABILITY.md) | 数据库独立部署与非阻断连接：对应主题的当前使用、实现或验证说明（docs/DATABASE_AVAILABILITY.md）。 |
| [docs/DEAD_CODE_AUDIT.md](../docs/DEAD_CODE_AUDIT.md) | 死代码清理与验收记录：保留的历史决定或验收记录，请以 2.0.2 当前指南为准（docs/DEAD_CODE_AUDIT.md）。 |
| [docs/FILES.md](../docs/FILES.md) | 当前版本逐文件中文职责索引，供阅读查找（docs/FILES.md）。 |
| [docs/HANDOFF.md](../docs/HANDOFF.md) | 2.0.2 维护交接：对应主题的当前使用、实现或验证说明（docs/HANDOFF.md）。 |
| [docs/LINUX.md](../docs/LINUX.md) | Linux 软件与数据库管理工具：对应主题的当前使用、实现或验证说明（docs/LINUX.md）。 |
| [docs/P0_DEVICE_SECRET_STORAGE.md](../docs/P0_DEVICE_SECRET_STORAGE.md) | 当前设备凭据边界：对应主题的当前使用、实现或验证说明（docs/P0_DEVICE_SECRET_STORAGE.md）。 |
| [docs/P0_RELATIONAL_INTEGRITY.md](../docs/P0_RELATIONAL_INTEGRITY.md) | 当前设备归属完整性：对应主题的当前使用、实现或验证说明（docs/P0_RELATIONAL_INTEGRITY.md）。 |
| [docs/PREVIEW_ISSUE_REMEDIATION.md](../docs/PREVIEW_ISSUE_REMEDIATION.md) | preview 问题清单核验与修复：保留的历史决定或验收记录，请以 2.0.2 当前指南为准（docs/PREVIEW_ISSUE_REMEDIATION.md）。 |
| [docs/PUBLICATION.md](../docs/PUBLICATION.md) | 2.0.2 交付说明：对应主题的当前使用、实现或验证说明（docs/PUBLICATION.md）。 |
| [docs/RELEASE.md](../docs/RELEASE.md) | 2.0.2 发布流程：对应主题的当前使用、实现或验证说明（docs/RELEASE.md）。 |
| [docs/SETUP.md](../docs/SETUP.md) | 使用与数据库配置 · 2.0.2：对应主题的当前使用、实现或验证说明（docs/SETUP.md）。 |
| [docs/SUPPORT.md](../docs/SUPPORT.md) | 中英支持与反馈：快速排查、可复制缺陷及功能模板和隐私边界（docs/SUPPORT.md）。 |
| [docs/TESTING.md](../docs/TESTING.md) | 测试分层 · 2.0.2：对应主题的当前使用、实现或验证说明（docs/TESTING.md）。 |
| [docs/UI_UX.md](../docs/UI_UX.md) | UI / UX 设计与验收契约：对应主题的当前使用、实现或验证说明（docs/UI_UX.md）。 |
| [docs/VALIDATION.md](../docs/VALIDATION.md) | 已发布 2.0.2 的历史本地验收记录：对应主题的当前使用、实现或验证说明（docs/VALIDATION.md）。 |
| [docs/decisions/0001-desktop-local-service-architecture.md](../docs/decisions/0001-desktop-local-service-architecture.md) | ADR-0001：Electron + 本机 Spring 服务：保留的历史决定或验收记录，请以 2.0.2 当前指南为准（docs/decisions/0001-desktop-local-service-architecture.md）。 |
| [docs/decisions/0002-local-h2-data-storage.md](../docs/decisions/0002-local-h2-data-storage.md) | ADR-0002：H2 文件数据库与用户数据目录：保留的历史决定或验收记录，请以 2.0.2 当前指南为准（docs/decisions/0002-local-h2-data-storage.md）。 |
| [docs/decisions/0003-electron-security-boundary.md](../docs/decisions/0003-electron-security-boundary.md) | ADR-0003：Electron Renderer/Main 安全边界：保留的历史决定或验收记录，请以 2.0.2 当前指南为准（docs/decisions/0003-electron-security-boundary.md）。 |
| [docs/decisions/0004-ui-design-system.md](../docs/decisions/0004-ui-design-system.md) | ADR-0004：面向老年照护场景的 UI 设计契约：保留的历史决定或验收记录，请以 2.0.2 当前指南为准（docs/decisions/0004-ui-design-system.md）。 |
| [docs/decisions/0005-preview-release-model.md](../docs/decisions/0005-preview-release-model.md) | ADR-0005：preview 预发布与交付模型：保留的历史决定或验收记录，请以 2.0.2 当前指南为准（docs/decisions/0005-preview-release-model.md）。 |
| [docs/decisions/0006-tauri-go-postgres.md](../docs/decisions/0006-tauri-go-postgres.md) | ADR-0006：Tauri / Go / 独立 PostgreSQL：对应主题的当前使用、实现或验证说明（docs/decisions/0006-tauri-go-postgres.md）。 |
| [docs/decisions/README.md](../docs/decisions/README.md) | 架构决定：对应主题的当前使用、实现或验证说明（docs/decisions/README.md）。 |
| [docs/file-descriptions.json](../docs/file-descriptions.json) | 当前版本逐文件中文职责索引，供工具与发布校验使用（docs/file-descriptions.json）。 |
| [docs/preview-remediation-validation.json](../docs/preview-remediation-validation.json) | 项目文件 docs/preview-remediation-validation.json 的实现与配置（docs/preview-remediation-validation.json）。 |
| [docs/releases/2.0.1.md](../docs/releases/2.0.1.md) | 岁宁 2.0.1 Preview：保留的历史决定或验收记录，请以 2.0.2 当前指南为准（docs/releases/2.0.1.md）。 |
| [docs/releases/2.0.2.md](../docs/releases/2.0.2.md) | 岁宁 2.0.2 · 更新安装包与 Linux 附件：对应主题的当前使用、实现或验证说明（docs/releases/2.0.2.md）。 |
| [docs/runbooks/backup-restore.md](../docs/runbooks/backup-restore.md) | 2.0.2 备份与恢复：对应主题的当前使用、实现或验证说明（docs/runbooks/backup-restore.md）。 |
| [docs/runbooks/release.md](../docs/runbooks/release.md) | 2.0.2 发布操作：对应主题的当前使用、实现或验证说明（docs/runbooks/release.md）。 |
| [docs/runbooks/rollback.md](../docs/runbooks/rollback.md) | 2.0.2 回退：对应主题的当前使用、实现或验证说明（docs/runbooks/rollback.md）。 |
| [docs/runbooks/signing.md](../docs/runbooks/signing.md) | Windows 签名：对应主题的当前使用、实现或验证说明（docs/runbooks/signing.md）。 |
| [docs/screenshots/family-mobile.png](../docs/screenshots/family-mobile.png) | 打包软件家属窄屏实拍：照护入口、公告与不遮挡内容的底部导航。（docs/screenshots/family-mobile.png）。 |
| [docs/screenshots/family-screen-fit-20261016.png](../docs/screenshots/family-screen-fit-20261016.png) | 1366×768 家属照护首页截图，展示更紧凑的首屏布局、公告与常用记录。（docs/screenshots/family-screen-fit-20261016.png）。 |
| [docs/screenshots/invite-code-desktop.png](../docs/screenshots/invite-code-desktop.png) | 打包软件邀请码管理实拍：管理员检索与管理家属注册邀请码。（docs/screenshots/invite-code-desktop.png）。 |
| [docs/screenshots/login-desktop.png](../docs/screenshots/login-desktop.png) | 桌面登录实拍：山水背景、原选定 Logo 与手机号密码表单。（docs/screenshots/login-desktop.png）。 |
| [docs/screenshots/login-screen-fit-20261016.png](../docs/screenshots/login-screen-fit-20261016.png) | 1366×768 登录页生产构建截图，展示一屏布局、湖岸画面与完整登录操作。（docs/screenshots/login-screen-fit-20261016.png）。 |
| [docs/screenshots/login-short-screen-20261016.png](../docs/screenshots/login-short-screen-20261016.png) | 360×560 短屏登录真实截图，展示收起辅助文案后完整可达的表单与注册入口。（docs/screenshots/login-short-screen-20261016.png）。 |
| [docs/screenshots/overview-desktop.png](../docs/screenshots/overview-desktop.png) | 管理员总览实拍：测试数据下的异常类别、告警和平台公告。（docs/screenshots/overview-desktop.png）。 |
| [docs/screenshots/setup-desktop.png](../docs/screenshots/setup-desktop.png) | 打包软件首次设置实拍：创建本机管理员，展示无默认账号的启动流程。（docs/screenshots/setup-desktop.png）。 |
| [docs/screenshots/tauri-2.0.2-main.png](../docs/screenshots/tauri-2.0.2-main.png) | 2.0.2 实际安装后的 Tauri WebView 截图：真实登录后的管理员运营总览，使用临时测试库（docs/screenshots/tauri-2.0.2-main.png）。 |
| [docs/screenshots/tauri-2.0.2-offline.png](../docs/screenshots/tauri-2.0.2-offline.png) | 2.0.2 实际安装后的 Tauri WebView 截图：数据库离线连接状态（docs/screenshots/tauri-2.0.2-offline.png）。 |
| [docs/verification.json](../docs/verification.json) | 此次预览版的测试数量、系统环境、签名状态和审计边界摘要。（docs/verification.json）。 |
| [tests/ui/ui-smoke.mjs](../tests/ui/ui-smoke.mjs) | 项目文件 tests/ui/ui-smoke.mjs 的实现与配置（tests/ui/ui-smoke.mjs）。 |
| [智慧养老手表管理系统/.gitignore](../智慧养老手表管理系统/.gitignore) | 桌面工程的生成目录、数据库、日志和私钥排除配置。（智慧养老手表管理系统/.gitignore）。 |
| [智慧养老手表管理系统/LICENSE](../智慧养老手表管理系统/LICENSE) | 桌面源码工程副本的 MIT 许可，方便独立源码包保留授权。（智慧养老手表管理系统/LICENSE）。 |
| [智慧养老手表管理系统/README.md](../智慧养老手表管理系统/README.md) | 桌面源码工程的架构、数据目录、构建与运行边界说明。（智慧养老手表管理系统/README.md）。 |
| [智慧养老手表管理系统/backend/README.md](../智慧养老手表管理系统/backend/README.md) | Go 后端配置、独立 PostgreSQL、迁移与离线启动说明（智慧养老手表管理系统/backend/README.md）。 |
| [智慧养老手表管理系统/backend/VERSION](../智慧养老手表管理系统/backend/VERSION) | Go 后端对应的产品版本（智慧养老手表管理系统/backend/VERSION）。 |
| [智慧养老手表管理系统/backend/cmd/migrate/main.go](../智慧养老手表管理系统/backend/cmd/migrate/main.go) | Go 模块 migrate：main（智慧养老手表管理系统/backend/cmd/migrate/main.go）。 |
| [智慧养老手表管理系统/backend/cmd/server/database.go](../智慧养老手表管理系统/backend/cmd/server/database.go) | Go 模块 server：middleware、watchDatabase（智慧养老手表管理系统/backend/cmd/server/database.go）。 |
| [智慧养老手表管理系统/backend/cmd/server/database_test.go](../智慧养老手表管理系统/backend/cmd/server/database_test.go) | Go 模块 server：TestDatabaseUnavailableDoesNotBlockApplication、TestLogoutDoesNotRequireDatabase（智慧养老手表管理系统/backend/cmd/server/database_test.go）。 |
| [智慧养老手表管理系统/backend/cmd/server/desktop.go](../智慧养老手表管理系统/backend/cmd/server/desktop.go) | Go 模块 server：applyDesktopEnv、writeReadyFile、watchParent（智慧养老手表管理系统/backend/cmd/server/desktop.go）。 |
| [智慧养老手表管理系统/backend/cmd/server/main.go](../智慧养老手表管理系统/backend/cmd/server/main.go) | Go 模块 server：main（智慧养老手表管理系统/backend/cmd/server/main.go）。 |
| [智慧养老手表管理系统/backend/cmd/server/parentwatch_unix.go](../智慧养老手表管理系统/backend/cmd/server/parentwatch_unix.go) | Go 模块 server：parentAlive（智慧养老手表管理系统/backend/cmd/server/parentwatch_unix.go）。 |
| [智慧养老手表管理系统/backend/cmd/server/parentwatch_windows.go](../智慧养老手表管理系统/backend/cmd/server/parentwatch_windows.go) | Go 模块 server：parentAlive（智慧养老手表管理系统/backend/cmd/server/parentwatch_windows.go）。 |
| [智慧养老手表管理系统/backend/cmd/server/regression_test.go](../智慧养老手表管理系统/backend/cmd/server/regression_test.go) | 已审查问题的数据库契约、安全与竞争回归（智慧养老手表管理系统/backend/cmd/server/regression_test.go）。 |
| [智慧养老手表管理系统/backend/go.mod](../智慧养老手表管理系统/backend/go.mod) | Go 后端直接与间接模块依赖及语言最低版本（智慧养老手表管理系统/backend/go.mod）。 |
| [智慧养老手表管理系统/backend/go.sum](../智慧养老手表管理系统/backend/go.sum) | Go 模块内容与版本校验摘要（智慧养老手表管理系统/backend/go.sum）。 |
| [智慧养老手表管理系统/backend/internal/auth/auth.go](../智慧养老手表管理系统/backend/internal/auth/auth.go) | 角色授权、公开路径边界与密码验证（智慧养老手表管理系统/backend/internal/auth/auth.go）。 |
| [智慧养老手表管理系统/backend/internal/auth/password.go](../智慧养老手表管理系统/backend/internal/auth/password.go) | Go 模块 auth：Hash、Matches、ValidNewPassword、ValidPhone（智慧养老手表管理系统/backend/internal/auth/password.go）。 |
| [智慧养老手表管理系统/backend/internal/auth/regression_test.go](../智慧养老手表管理系统/backend/internal/auth/regression_test.go) | 已审查问题的数据库契约、安全与竞争回归（智慧养老手表管理系统/backend/internal/auth/regression_test.go）。 |
| [智慧养老手表管理系统/backend/internal/config/config.go](../智慧养老手表管理系统/backend/internal/config/config.go) | 服务地址、会话、图片目录与数据库环境变量配置（智慧养老手表管理系统/backend/internal/config/config.go）。 |
| [智慧养老手表管理系统/backend/internal/csrf/csrf.go](../智慧养老手表管理系统/backend/internal/csrf/csrf.go) | 会话 CSRF 校验与设备/原生令牌接口豁免（智慧养老手表管理系统/backend/internal/csrf/csrf.go）。 |
| [智慧养老手表管理系统/backend/internal/db/db.go](../智慧养老手表管理系统/backend/internal/db/db.go) | PostgreSQL 连接池、迁移版本与事务串行化（智慧养老手表管理系统/backend/internal/db/db.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/handler_auth.go](../智慧养老手表管理系统/backend/internal/handlers/handler_auth.go) | 管理员初始化、会话登录、退出及账号安全接口（智慧养老手表管理系统/backend/internal/handlers/handler_auth.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/handler_family.go](../智慧养老手表管理系统/backend/internal/handlers/handler_family.go) | 家属账号、邀请码领取与长者关联管理（智慧养老手表管理系统/backend/internal/handlers/handler_family.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/handler_old.go](../智慧养老手表管理系统/backend/internal/handlers/handler_old.go) | 长者档案、关联检查和删除约束（智慧养老手表管理系统/backend/internal/handlers/handler_old.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/handler_social.go](../智慧养老手表管理系统/backend/internal/handlers/handler_social.go) | 平台公告、生活记录与互动接口（智慧养老手表管理系统/backend/internal/handlers/handler_social.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/handler_user.go](../智慧养老手表管理系统/backend/internal/handlers/handler_user.go) | 用户档案与角色权限管理接口（智慧养老手表管理系统/backend/internal/handlers/handler_user.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go](../智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go) | 健康数据、设备上报、位置与求助接口（智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/handlers.go](../智慧养老手表管理系统/backend/internal/handlers/handlers.go) | 业务路由注册、共享处理器与依赖装配（智慧养老手表管理系统/backend/internal/handlers/handlers.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/integration_test.go](../智慧养老手表管理系统/backend/internal/handlers/integration_test.go) | 专用 PostgreSQL 上的鉴权、事务及业务集成回归（智慧养老手表管理系统/backend/internal/handlers/integration_test.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/regression_test.go](../智慧养老手表管理系统/backend/internal/handlers/regression_test.go) | 已审查问题的数据库契约、安全与竞争回归（智慧养老手表管理系统/backend/internal/handlers/regression_test.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/util.go](../智慧养老手表管理系统/backend/internal/handlers/util.go) | 业务请求解析、参数验证与处理器共享辅助函数（智慧养老手表管理系统/backend/internal/handlers/util.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/web.go](../智慧养老手表管理系统/backend/internal/handlers/web.go) | 内嵌 Vue 静态资源挂载及浏览器安全响应头（智慧养老手表管理系统/backend/internal/handlers/web.go）。 |
| [智慧养老手表管理系统/backend/internal/handlers/web/.gitignore](../智慧养老手表管理系统/backend/internal/handlers/web/.gitignore) | Go 模块 web：.gitignore（智慧养老手表管理系统/backend/internal/handlers/web/.gitignore）。 |
| [智慧养老手表管理系统/backend/internal/handlers/web/.gitkeep](../智慧养老手表管理系统/backend/internal/handlers/web/.gitkeep) | 仅保留内嵌前端目录，桌面构建时填入真实 Vue 产物（智慧养老手表管理系统/backend/internal/handlers/web/.gitkeep）。 |
| [智慧养老手表管理系统/backend/internal/ratelimit/ratelimit.go](../智慧养老手表管理系统/backend/internal/ratelimit/ratelimit.go) | 公开接口的请求限流与窗口清理（智慧养老手表管理系统/backend/internal/ratelimit/ratelimit.go）。 |
| [智慧养老手表管理系统/backend/internal/response/response.go](../智慧养老手表管理系统/backend/internal/response/response.go) | JSON 响应包络与通用错误输出（智慧养老手表管理系统/backend/internal/response/response.go）。 |
| [智慧养老手表管理系统/backend/internal/session/session.go](../智慧养老手表管理系统/backend/internal/session/session.go) | 内存会话、过期清理与 Cookie 生命周期（智慧养老手表管理系统/backend/internal/session/session.go）。 |
| [智慧养老手表管理系统/backend/migrations/0001_init.sql](../智慧养老手表管理系统/backend/migrations/0001_init.sql) | PostgreSQL 0001_init 数据结构迁移与约束定义（智慧养老手表管理系统/backend/migrations/0001_init.sql）。 |
| [智慧养老手表管理系统/backend/migrations/0002_daily_uniqueness.sql](../智慧养老手表管理系统/backend/migrations/0002_daily_uniqueness.sql) | PostgreSQL 0002_daily_uniqueness 数据结构迁移与约束定义（智慧养老手表管理系统/backend/migrations/0002_daily_uniqueness.sql）。 |
| [智慧养老手表管理系统/backend/migrations/0003_alarm_process.sql](../智慧养老手表管理系统/backend/migrations/0003_alarm_process.sql) | PostgreSQL 0003_alarm_process 数据结构迁移与约束定义（智慧养老手表管理系统/backend/migrations/0003_alarm_process.sql）。 |
| [智慧养老手表管理系统/backend/migrations/0004_device_oldperson_fk.sql](../智慧养老手表管理系统/backend/migrations/0004_device_oldperson_fk.sql) | PostgreSQL 0004_device_oldperson_fk 数据结构迁移与约束定义（智慧养老手表管理系统/backend/migrations/0004_device_oldperson_fk.sql）。 |
| [智慧养老手表管理系统/backend/migrations/embed.go](../智慧养老手表管理系统/backend/migrations/embed.go) | 将迁移 SQL 编译为不依赖工作目录的嵌入文件系统（智慧养老手表管理系统/backend/migrations/embed.go）。 |
| [智慧养老手表管理系统/desktop/.gitignore](../智慧养老手表管理系统/desktop/.gitignore) | 桌面 .gitignore 配置与资源（智慧养老手表管理系统/desktop/.gitignore）。 |
| [智慧养老手表管理系统/desktop/Cargo.lock](../智慧养老手表管理系统/desktop/Cargo.lock) | Tauri/Rust 精确依赖版本及来源校验锁定（智慧养老手表管理系统/desktop/Cargo.lock）。 |
| [智慧养老手表管理系统/desktop/Cargo.toml](../智慧养老手表管理系统/desktop/Cargo.toml) | Rust 桌面工程依赖、发布优化与岁宁产品版本（智慧养老手表管理系统/desktop/Cargo.toml）。 |
| [智慧养老手表管理系统/desktop/bootstrap/index.html](../智慧养老手表管理系统/desktop/bootstrap/index.html) | 桌面 bootstrap\index.html 配置与资源（智慧养老手表管理系统/desktop/bootstrap/index.html）。 |
| [智慧养老手表管理系统/desktop/build.ps1](../智慧养老手表管理系统/desktop/build.ps1) | 前端嵌入、Go 回归、原生编译及 NSIS 安装包构建（智慧养老手表管理系统/desktop/build.ps1）。 |
| [智慧养老手表管理系统/desktop/build.rs](../智慧养老手表管理系统/desktop/build.rs) | Tauri 编译入口与后端哈希变更跟踪（智慧养老手表管理系统/desktop/build.rs）。 |
| [智慧养老手表管理系统/desktop/capabilities/desktop.json](../智慧养老手表管理系统/desktop/capabilities/desktop.json) | 仅主窗口及本机页面可用的最小 Tauri 权限集合（智慧养老手表管理系统/desktop/capabilities/desktop.json）。 |
| [智慧养老手表管理系统/desktop/package-lock.json](../智慧养老手表管理系统/desktop/package-lock.json) | 桌面 package-lock.json 配置与资源（智慧养老手表管理系统/desktop/package-lock.json）。 |
| [智慧养老手表管理系统/desktop/package.json](../智慧养老手表管理系统/desktop/package.json) | 原生应用名称、版本、主进程入口和项目许可元数据。（智慧养老手表管理系统/desktop/package.json）。 |
| [智慧养老手表管理系统/desktop/permissions/setup.toml](../智慧养老手表管理系统/desktop/permissions/setup.toml) | 仅授权管理员初始化命令的 Tauri 自定义权限（智慧养老手表管理系统/desktop/permissions/setup.toml）。 |
| [智慧养老手表管理系统/desktop/sign.ps1](../智慧养老手表管理系统/desktop/sign.ps1) | 正式证书或本机自签名 Authenticode 签署与状态记录（智慧养老手表管理系统/desktop/sign.ps1）。 |
| [智慧养老手表管理系统/desktop/src/main.rs](../智慧养老手表管理系统/desktop/src/main.rs) | 原生窗口、sidecar 完整性与生命周期及安全初始化命令（智慧养老手表管理系统/desktop/src/main.rs）。 |
| [智慧养老手表管理系统/desktop/suining.ico](../智慧养老手表管理系统/desktop/suining.ico) | 由用户选定山水 Logo 转换的多尺寸 Windows 程序和安装器图标。（智慧养老手表管理系统/desktop/suining.ico）。 |
| [智慧养老手表管理系统/desktop/tauri.conf.json](../智慧养老手表管理系统/desktop/tauri.conf.json) | Tauri 产品信息、sidecar、按需下载与静默安装 WebView2 及 NSIS 设置（智慧养老手表管理系统/desktop/tauri.conf.json）。 |
| [智慧养老手表管理系统/desktop/validate-version.ps1](../智慧养老手表管理系统/desktop/validate-version.ps1) | 核对前端、后端、Cargo 与 Tauri 的产品版本一致性（智慧养老手表管理系统/desktop/validate-version.ps1）。 |
| [智慧养老手表管理系统/desktop/version.json](../智慧养老手表管理系统/desktop/version.json) | 桌面 version.json 配置与资源（智慧养老手表管理系统/desktop/version.json）。 |
| [智慧养老手表管理系统/desktop/使用说明.txt](../智慧养老手表管理系统/desktop/使用说明.txt) | 随软件提供的中文操作指南，涵盖窗口全屏快捷键、首次账号、数据备份、设备联网和签名边界。（智慧养老手表管理系统/desktop/使用说明.txt）。 |
| [智慧养老手表管理系统/oldIot/README.md](../智慧养老手表管理系统/oldIot/README.md) | 前端开发命令、目录和本地素材使用说明。（智慧养老手表管理系统/oldIot/README.md）。 |
| [智慧养老手表管理系统/oldIot/index.html](../智慧养老手表管理系统/oldIot/index.html) | 网页入口：中文标题、图标、本地字体预加载、挂载点与 Vite 模块入口。（智慧养老手表管理系统/oldIot/index.html）。 |
| [智慧养老手表管理系统/oldIot/package-lock.json](../智慧养老手表管理系统/oldIot/package-lock.json) | 锁定当前前端安装树及摘要，移除仅供 Less 使用的可选依赖，保留其余版本。（智慧养老手表管理系统/oldIot/package-lock.json）。 |
| [智慧养老手表管理系统/oldIot/package.json](../智慧养老手表管理系统/oldIot/package.json) | 精简后的前端依赖与 Vite 命令，使用普通 CSS，不再声明 Less 预处理器。（智慧养老手表管理系统/oldIot/package.json）。 |
| [智慧养老手表管理系统/oldIot/public/static/.gitkeep](../智慧养老手表管理系统/oldIot/public/static/.gitkeep) | 保留静态资源根目录的占位文件，不包含用户上传内容。（智慧养老手表管理系统/oldIot/public/static/.gitkeep）。 |
| [智慧养老手表管理系统/oldIot/public/static/brand/suining-apple-touch-180.png](../智慧养老手表管理系统/oldIot/public/static/brand/suining-apple-touch-180.png) | 移动设备添加到主屏幕使用的 180 像素图标。（智慧养老手表管理系统/oldIot/public/static/brand/suining-apple-touch-180.png）。 |
| [智慧养老手表管理系统/oldIot/public/static/brand/suining-favicon-32.png](../智慧养老手表管理系统/oldIot/public/static/brand/suining-favicon-32.png) | 浏览器标签页使用的 32 像素品牌图标。（智慧养老手表管理系统/oldIot/public/static/brand/suining-favicon-32.png）。 |
| [智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-192.webp](../智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-192.webp) | 高像素密度页眉和 README 展示使用的 192 像素 Logo。（智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-192.webp）。 |
| [智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-512.webp](../智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-512.webp) | 首次设置等较大展示使用的 512 像素 Logo。（智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-512.webp）。 |
| [智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-96.webp](../智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-96.webp) | 页眉与较小展示使用的 96 像素山水 Logo。（智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-96.webp）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/care-welcome-serif-OFL.txt](../智慧养老手表管理系统/oldIot/public/static/fonts/care-welcome-serif-OFL.txt) | care-welcome-serif-OFL.txt 字体配套说明，记录原字体来源、子集或许可要求。（智慧养老手表管理系统/oldIot/public/static/fonts/care-welcome-serif-OFL.txt）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/font-notices.md](../智慧养老手表管理系统/oldIot/public/static/fonts/font-notices.md) | font-notices.md 字体配套说明，记录原字体来源、子集或许可要求。（智慧养老手表管理系统/oldIot/public/static/fonts/font-notices.md）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-LICENSE.txt](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-LICENSE.txt) | source-han-serif-LICENSE.txt 字体配套说明，记录原字体来源、子集或许可要求。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-LICENSE.txt）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-000.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-000.woff2) | 中文衬线字体分片 000，覆盖 U+2E80-2E99,U+2E9B-2EF3,U+2F00-2FD5,U+2FF0-2FFB,U+3003-300B,U+300E-303F,U+3041-3096,U+3099-30B2…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-000.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-001.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-001.woff2) | 中文衬线字体分片 001，覆盖 U+30B3-30FF,U+3105-312F,U+3131-318E,U+3190-31BB,U+31C0-31E3,U+31F0-321E,U+3220-32CA…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-001.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-002.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-002.woff2) | 中文衬线字体分片 002，覆盖 U+32CB-332B,U+332D-34CB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-002.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-003.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-003.woff2) | 中文衬线字体分片 003，覆盖 U+34CC-36CB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-003.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-004.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-004.woff2) | 中文衬线字体分片 004，覆盖 U+36CC-38CB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-004.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-005.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-005.woff2) | 中文衬线字体分片 005，覆盖 U+38CC-3ACB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-005.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-006.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-006.woff2) | 中文衬线字体分片 006，覆盖 U+3ACC-3CCB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-006.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-007.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-007.woff2) | 中文衬线字体分片 007，覆盖 U+3CCC-3ECB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-007.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-008.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-008.woff2) | 中文衬线字体分片 008，覆盖 U+3ECC-40CB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-008.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-009.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-009.woff2) | 中文衬线字体分片 009，覆盖 U+40CC-42CB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-009.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-010.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-010.woff2) | 中文衬线字体分片 010，覆盖 U+42CC-44CB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-010.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-011.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-011.woff2) | 中文衬线字体分片 011，覆盖 U+44CC-46CB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-011.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-012.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-012.woff2) | 中文衬线字体分片 012，覆盖 U+46CC-48CB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-012.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-013.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-013.woff2) | 中文衬线字体分片 013，覆盖 U+48CC-4ACB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-013.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-014.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-014.woff2) | 中文衬线字体分片 014，覆盖 U+4ACC-4CCB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-014.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-015.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-015.woff2) | 中文衬线字体分片 015，覆盖 U+4CCC-4DBF,U+4E01-4E09,U+4E0C,U+4E0F-4E12,U+4E14-4E21,U+4E23,U+4E25-4E29,U+4E2B-4E2C,U+4E2E-4E…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-015.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-016.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-016.woff2) | 中文衬线字体分片 016，覆盖 U+4F31-4F33,U+4F35-4F37,U+4F39-4F45,U+4F47-4F4C,U+4F4F-4F52,U+4F54-4F58,U+4F5A-4F5B,U+4F5D-4F5F…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-016.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-017.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-017.woff2) | 中文衬线字体分片 017，覆盖 U+514E-5164,U+5166-5167,U+5169-516B,U+516D-5170,U+5172,U+5174-5175,U+5178-517A,U+517C-5184,U+51…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-017.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-018.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-018.woff2) | 中文衬线字体分片 018，覆盖 U+537A-5385,U+5387-538A,U+538C-539E,U+53A0-53BA,U+53BC-53C9,U+53CB-53CC,U+53CE-53D0,U+53D2-53D5…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-018.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-019.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-019.woff2) | 中文衬线字体分片 019，覆盖 U+559E-56DD,U+56DF,U+56E1-56F3,U+56F5-56FD,U+56FF-5727,U+5729-572F,U+5731-573F,U+5741-574F,U+57…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-019.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-020.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-020.woff2) | 中文衬线字体分片 020，覆盖 U+57A7-57F9,U+57FB-586A,U+586C-5903,U+5905-5906,U+5908-590C,U+590E-5919,U+591B-5926,U+5928,U+59…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-020.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-021.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-021.woff2) | 中文衬线字体分片 021，覆盖 U+59B5-59CA,U+59CC-59D2,U+59D4-5B4F,U+5B51-5B56,U+5B59-5B87,U+5B8A-5B8B,U+5B8D-5B99,U+5B9B-5B9D…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-021.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-022.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-022.woff2) | 中文衬线字体分片 022，覆盖 U+5BC1-5BC5,U+5BC7-5BDE,U+5BE0-5BF8,U+5BFA-5BFB,U+5BFD-5C05,U+5C07-5C0E,U+5C10,U+5C12-5C19,U+5C…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-022.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-023.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-023.woff2) | 中文衬线字体分片 023，覆盖 U+5DCD-5DE4,U+5DE7-5DF0,U+5DF3-5E02,U+5E04-5E25,U+5E27-5E37,U+5E39-5E72,U+5E75,U+5E77-5E8E,U+5E…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-023.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-024.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-024.woff2) | 中文衬线字体分片 024，覆盖 U+5FED-5FFF,U+6002-600D,U+600F-6026,U+6028-603A,U+603C-606E,U+6070-6088,U+608A-60A7,U+60A9-60C4…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-024.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-025.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-025.woff2) | 中文衬线字体分片 025，覆盖 U+61F9-620F,U+6212-6215,U+6217-6233,U+6235-6236,U+6238-623E,U+6241-624A,U+624C-6252,U+6254-627D…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-025.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-026.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-026.woff2) | 中文衬线字体分片 026，覆盖 U+6416-641B,U+641D-6453,U+6455-6457,U+6459-64CC,U+64CE-652E,U+6530-6535,U+6537-6538,U+653A-653D…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-026.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-027.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-027.woff2) | 中文衬线字体分片 027，覆盖 U+662C-662E,U+6630-663D,U+663F-6667,U+6669-666F,U+6671-6679,U+667B-6681,U+6683-66F1,U+66F3,U+66…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-027.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-028.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-028.woff2) | 中文衬线字体分片 028，覆盖 U+684C-6862,U+6864-68BF,U+68C1-6959,U+695B-6A20,U+6A22-6A4F…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-028.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-029.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-029.woff2) | 中文衬线字体分片 029，覆盖 U+6A50-6B20,U+6B23-6B62,U+6B66-6BCE,U+6BD0-6BDA,U+6BDC-6C13,U+6C15-6C33,U+6C35-6C37,U+6C39-6C59…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-029.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-030.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-030.woff2) | 中文衬线字体分片 030，覆盖 U+6C5A-6CA0,U+6CA2-6CD4,U+6CD6-6CE7,U+6CE9-6D00,U+6D02-6D11,U+6D13-6D3A,U+6D3C-6D40,U+6D42-6D4A…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-030.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-031.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-031.woff2) | 中文衬线字体分片 031，覆盖 U+6E67-6E8F,U+6E91-6ED0,U+6ED2-6FBF,U+6FC1-7069…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-031.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-032.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-032.woff2) | 中文衬线字体分片 032，覆盖 U+706A-70B8,U+70BA-7135,U+7137-7166,U+7168-7230,U+7232-7236,U+7238-7246,U+7249-7268,U+726A-7271…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-032.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-033.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-033.woff2) | 中文衬线字体分片 033，覆盖 U+7272-7274,U+7276-72B5,U+72B7-72EB,U+72ED-7386,U+7388-73AE,U+73B1-7405,U+7407-7478…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-033.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-034.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-034.woff2) | 中文衬线字体分片 034，覆盖 U+7479-751E,U+7520-7527,U+7529-7530,U+7532-7534,U+7536,U+7538-7558,U+755A-75C4,U+75C6-767A,U+76…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-034.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-035.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-035.woff2) | 中文衬线字体分片 035，覆盖 U+7681-7683,U+7685-76D0,U+76D2-76ED,U+76EF-76F3,U+76F5-76F7,U+76F9-770A,U+770C-771F,U+7721-7760…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-035.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-036.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-036.woff2) | 中文衬线字体分片 036，覆盖 U+788E-7939,U+793B-79BA,U+79BC-79D1,U+79D3-79EF,U+79F1-79FA,U+79FC-7A0C,U+7A0E-7A79,U+7A7B-7A94…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-036.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-037.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-037.woff2) | 中文衬线字体分片 037，覆盖 U+7A95-7A96,U+7A98-7ACA,U+7ACC-7ADF,U+7AE1-7AEE,U+7AF0-7B25,U+7B27-7B2B,U+7B2D-7B53,U+7B55-7B5A…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-037.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-038.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-038.woff2) | 中文衬线字体分片 038，覆盖 U+7CA0-7CC9,U+7CCB-7CFA,U+7CFC-7D21,U+7D23-7EA1,U+7EA3…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-038.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-039.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-039.woff2) | 中文衬线字体分片 039，覆盖 U+7EA4-7EAB,U+7EAD-7EB4,U+7EB6-7EB9,U+7EBB-7EBE,U+7EC0-7EC5,U+7EC7,U+7EC9-7ECE,U+7ED0,U+7ED2,U+…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-039.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-040.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-040.woff2) | 中文衬线字体分片 040，覆盖 U+80C0-80FC,U+80FE-81E9,U+81EB-81F2,U+81F5-8229,U+822B,U+822D-8271,U+8273-8281,U+8283-82C7…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-040.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-041.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-041.woff2) | 中文衬线字体分片 041，覆盖 U+82C8-8302,U+8304-836E,U+8370-83B6,U+83B8-8424,U+8426-84CB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-041.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-042.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-042.woff2) | 中文衬线字体分片 042，覆盖 U+84CC-85CE,U+85D0-86CC…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-042.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-043.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-043.woff2) | 中文衬线字体分片 043，覆盖 U+86CD-883F,U+8841-884B,U+884D-8864,U+8866-8867,U+8869-886A,U+886D-889B,U+889D-88AA,U+88AC-88D4…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-043.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-044.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-044.woff2) | 中文衬线字体分片 044，覆盖 U+88D5-88E3,U+88E5-8980,U+8982-89C1,U+89C3-89C7,U+89C9-89D1,U+89D3-89E2,U+89E4-89FF,U+8A01-8ADB…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-044.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-045.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-045.woff2) | 中文衬线字体分片 045，覆盖 U+8ADC-8B65,U+8B67-8BA0,U+8BA2-8BA3,U+8BA5-8BA8,U+8BAA-8BAF,U+8BB1-8BB9,U+8BBB-8BBD,U+8BC0,U+8B…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-045.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-046.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-046.woff2) | 中文衬线字体分片 046，覆盖 U+8CF3-8D24,U+8D27-8D38,U+8D3A-8D43,U+8D45-8D5D,U+8D5F-8D76,U+8D78-8D84,U+8D86-8DDE,U+8DE0-8DEE…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-046.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-047.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-047.woff2) | 中文衬线字体分片 047，覆盖 U+8EFF-8F6B,U+8F6D-8F73,U+8F75-8F7C,U+8F7E-8F90,U+8F92,U+8F94-8FC6,U+8FC8-8FCD,U+8FCF,U+8FD2-8F…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-047.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-048.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-048.woff2) | 中文衬线字体分片 048，覆盖 U+9117-914C,U+914E-9191,U+9193-91CB,U+91CE,U+91D0-931B…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-048.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-049.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-049.woff2) | 中文衬线字体分片 049，覆盖 U+931C-949E,U+94A0-94A4,U+94A6-94FF,U+9501-951E…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-049.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-050.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-050.woff2) | 中文衬线字体分片 050，覆盖 U+951F-952D,U+952F-957E,U+9580-95E7,U+95E9-95EC,U+95EF-95F3,U+95F5-9604,U+9606-9607,U+9609-9643…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-050.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-051.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-051.woff2) | 中文衬线字体分片 051，覆盖 U+9733-9758,U+975A-9761,U+9763-978A,U+978C-9874,U+9877-9878,U+987A,U+987C-9883,U+9885-9897,U+98…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-051.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-052.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-052.woff2) | 中文衬线字体分片 052，覆盖 U+993E-9995,U+9997-9A8B,U+9A8D-9AD7,U+9AD9-9B40…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-052.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-053.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-053.woff2) | 中文衬线字体分片 053，覆盖 U+9B41-9D40…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-053.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-054.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-054.woff2) | 中文衬线字体分片 054，覆盖 U+9D41-9ED7,U+9ED9-9F41…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-054.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-055.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-055.woff2) | 中文衬线字体分片 055，覆盖 U+9F42-9F83,U+9F85-9FFF,U+A960-A97C,U+AC00-AD25…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-055.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-056.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-056.woff2) | 中文衬线字体分片 056，覆盖 U+AD26-AF25…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-056.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-057.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-057.woff2) | 中文衬线字体分片 057，覆盖 U+AF26-B125…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-057.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-058.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-058.woff2) | 中文衬线字体分片 058，覆盖 U+B126-B325…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-058.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-059.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-059.woff2) | 中文衬线字体分片 059，覆盖 U+B326-B525…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-059.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-060.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-060.woff2) | 中文衬线字体分片 060，覆盖 U+B526-B725…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-060.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-061.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-061.woff2) | 中文衬线字体分片 061，覆盖 U+B726-B925…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-061.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-062.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-062.woff2) | 中文衬线字体分片 062，覆盖 U+B926-BB25…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-062.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-063.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-063.woff2) | 中文衬线字体分片 063，覆盖 U+BB26-BD25…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-063.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-064.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-064.woff2) | 中文衬线字体分片 064，覆盖 U+BD26-BF25…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-064.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-065.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-065.woff2) | 中文衬线字体分片 065，覆盖 U+BF26-C125…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-065.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-066.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-066.woff2) | 中文衬线字体分片 066，覆盖 U+C126-C325…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-066.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-067.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-067.woff2) | 中文衬线字体分片 067，覆盖 U+C326-C525…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-067.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-068.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-068.woff2) | 中文衬线字体分片 068，覆盖 U+C526-C725…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-068.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-069.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-069.woff2) | 中文衬线字体分片 069，覆盖 U+C726-C925…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-069.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-070.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-070.woff2) | 中文衬线字体分片 070，覆盖 U+C926-CB25…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-070.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-071.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-071.woff2) | 中文衬线字体分片 071，覆盖 U+CB26-CD25…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-071.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-072.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-072.woff2) | 中文衬线字体分片 072，覆盖 U+CD26-CF25…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-072.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-073.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-073.woff2) | 中文衬线字体分片 073，覆盖 U+CF26-D125…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-073.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-074.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-074.woff2) | 中文衬线字体分片 074，覆盖 U+D126-D325…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-074.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-075.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-075.woff2) | 中文衬线字体分片 075，覆盖 U+D326-D525…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-075.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-076.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-076.woff2) | 中文衬线字体分片 076，覆盖 U+D526-D725…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-076.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-077.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-077.woff2) | 中文衬线字体分片 077，覆盖 U+D726-D7A3,U+D7B0-D7C6,U+D7CB-D7FB,U+F900-FA39…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-077.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-078.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-078.woff2) | 中文衬线字体分片 078，覆盖 U+FA3A-FA6D,U+FB00-FB04,U+FE10-FE19,U+FE30-FE52,U+FE54-FE66,U+FE68-FE6B,U+FF02-FF07,U+FF0A-FF0B…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-078.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-079.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-079.woff2) | 中文衬线字体分片 079，覆盖 U+1F202,U+1F210-1F23B,U+1F240-1F248,U+1F250-1F251,U+2000B,U+20021,U+2003E,U+20046,U+2004E,U+200…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-079.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-080.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-080.woff2) | 中文衬线字体分片 080，覆盖 U+21922,U+21927,U+2193B,U+21944,U+21958,U+2196A,U+2197C,U+21980,U+21983,U+21988,U+21996,U+219C3…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-080.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-081.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-081.woff2) | 中文衬线字体分片 081，覆盖 U+243F8,U+24404,U+24435-24436,U+2445A-2445B,U+24473,U+24487-24488,U+244B9,U+244BC,U+244CE,U+244…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-081.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-082.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-082.woff2) | 中文衬线字体分片 082，覆盖 U+26DAE,U+26DDC,U+26DEA-26DEB,U+26DF0,U+26E00,U+26E05,U+26E07,U+26E12,U+26E40,U+26E42-26E45,U+2…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-082.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-083.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-083.woff2) | 中文衬线字体分片 083，覆盖 U+29EE9,U+29EF8,U+29F23,U+29F30,U+29F7E,U+29F83,U+29F8C,U+29FB7,U+29FCE,U+29FD7,U+29FDE,U+2A014…；按所需字符加载。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/characters-083.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/manifest.json](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/manifest.json) | manifest.json 字体配套说明，记录原字体来源、子集或许可要求。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/manifest.json）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/ui-common.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/ui-common.woff2) | 中文界面常用字符字体分片，作为首屏预加载资源。（智慧养老手表管理系统/oldIot/public/static/fonts/source-han-serif-sc/ui-common.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-serif-4-italic.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-serif-4-italic.woff2) | source-serif-4-italic.woff2 字体配套说明，记录原字体来源、子集或许可要求。（智慧养老手表管理系统/oldIot/public/static/fonts/source-serif-4-italic.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-serif-4.woff2](../智慧养老手表管理系统/oldIot/public/static/fonts/source-serif-4.woff2) | Source Serif 4 英文与数字的本地可变字体，避免依赖外部字体网络。（智慧养老手表管理系统/oldIot/public/static/fonts/source-serif-4.woff2）。 |
| [智慧养老手表管理系统/oldIot/public/static/fonts/source-serif-LICENSE.md](../智慧养老手表管理系统/oldIot/public/static/fonts/source-serif-LICENSE.md) | source-serif-LICENSE.md 字体配套说明，记录原字体来源、子集或许可要求。（智慧养老手表管理系统/oldIot/public/static/fonts/source-serif-LICENSE.md）。 |
| [智慧养老手表管理系统/oldIot/public/static/icons/LICENSE.txt](../智慧养老手表管理系统/oldIot/public/static/icons/LICENSE.txt) | 保留图标字体对应的 Element UI 开源许可。（智慧养老手表管理系统/oldIot/public/static/icons/LICENSE.txt）。 |
| [智慧养老手表管理系统/oldIot/public/static/icons/element-icons.woff](../智慧养老手表管理系统/oldIot/public/static/icons/element-icons.woff) | 保留旧界面图标字形的本地 WOFF 资源，框架已切换 Element Plus。（智慧养老手表管理系统/oldIot/public/static/icons/element-icons.woff）。 |
| [智慧养老手表管理系统/oldIot/public/static/img/care-announcement-empty.webp](../智慧养老手表管理系统/oldIot/public/static/img/care-announcement-empty.webp) | 公告列表无记录时的温和空状态插图。（智慧养老手表管理系统/oldIot/public/static/img/care-announcement-empty.webp）。 |
| [智慧养老手表管理系统/oldIot/public/static/img/care-botanical.webp](../智慧养老手表管理系统/oldIot/public/static/img/care-botanical.webp) | 现有样式引用的植物装饰素材。（智慧养老手表管理系统/oldIot/public/static/img/care-botanical.webp）。 |
| [智慧养老手表管理系统/oldIot/public/static/img/care-courtyard.webp](../智慧养老手表管理系统/oldIot/public/static/img/care-courtyard.webp) | 照护页面使用的庭院风景插画。（智慧养老手表管理系统/oldIot/public/static/img/care-courtyard.webp）。 |
| [智慧养老手表管理系统/oldIot/public/static/img/care-lakeside-artisan-640.webp](../智慧养老手表管理系统/oldIot/public/static/img/care-lakeside-artisan-640.webp) | 湖畔风景的窄屏 srcset 版本，避免手机下载过大图片。（智慧养老手表管理系统/oldIot/public/static/img/care-lakeside-artisan-640.webp）。 |
| [智慧养老手表管理系统/oldIot/public/static/img/care-lakeside-artisan.webp](../智慧养老手表管理系统/oldIot/public/static/img/care-lakeside-artisan.webp) | 登录与家属页使用的本地湖畔风景主图。（智慧养老手表管理系统/oldIot/public/static/img/care-lakeside-artisan.webp）。 |
| [智慧养老手表管理系统/oldIot/public/static/img/care-river-morning.webp](../智慧养老手表管理系统/oldIot/public/static/img/care-river-morning.webp) | 现有页面样式引用的河岸晨景素材。（智慧养老手表管理系统/oldIot/public/static/img/care-river-morning.webp）。 |
| [智慧养老手表管理系统/oldIot/qa/.gitignore](../智慧养老手表管理系统/oldIot/qa/.gitignore) | 项目文件 智慧养老手表管理系统/oldIot/qa/.gitignore 的实现与配置（智慧养老手表管理系统/oldIot/qa/.gitignore）。 |
| [智慧养老手表管理系统/oldIot/qa/browser-acceptance.cjs](../智慧养老手表管理系统/oldIot/qa/browser-acceptance.cjs) | 项目文件 智慧养老手表管理系统/oldIot/qa/browser-acceptance.cjs 的实现与配置（智慧养老手表管理系统/oldIot/qa/browser-acceptance.cjs）。 |
| [智慧养老手表管理系统/oldIot/qa/browser-harness.cjs](../智慧养老手表管理系统/oldIot/qa/browser-harness.cjs) | 项目文件 智慧养老手表管理系统/oldIot/qa/browser-harness.cjs 的实现与配置（智慧养老手表管理系统/oldIot/qa/browser-harness.cjs）。 |
| [智慧养老手表管理系统/oldIot/qa/controls-acceptance.cjs](../智慧养老手表管理系统/oldIot/qa/controls-acceptance.cjs) | 项目文件 智慧养老手表管理系统/oldIot/qa/controls-acceptance.cjs 的实现与配置（智慧养老手表管理系统/oldIot/qa/controls-acceptance.cjs）。 |
| [智慧养老手表管理系统/oldIot/qa/database-availability-acceptance.cjs](../智慧养老手表管理系统/oldIot/qa/database-availability-acceptance.cjs) | 非阻断离线、自动恢复、权限边界与输入保留的浏览器回归（智慧养老手表管理系统/oldIot/qa/database-availability-acceptance.cjs）。 |
| [智慧养老手表管理系统/oldIot/qa/elder-delete-acceptance.cjs](../智慧养老手表管理系统/oldIot/qa/elder-delete-acceptance.cjs) | 项目文件 智慧养老手表管理系统/oldIot/qa/elder-delete-acceptance.cjs 的实现与配置（智慧养老手表管理系统/oldIot/qa/elder-delete-acceptance.cjs）。 |
| [智慧养老手表管理系统/oldIot/qa/font-hashes-before.json](../智慧养老手表管理系统/oldIot/qa/font-hashes-before.json) | 项目文件 智慧养老手表管理系统/oldIot/qa/font-hashes-before.json 的实现与配置（智慧养老手表管理系统/oldIot/qa/font-hashes-before.json）。 |
| [智慧养老手表管理系统/oldIot/qa/forms-acceptance.cjs](../智慧养老手表管理系统/oldIot/qa/forms-acceptance.cjs) | 项目文件 智慧养老手表管理系统/oldIot/qa/forms-acceptance.cjs 的实现与配置（智慧养老手表管理系统/oldIot/qa/forms-acceptance.cjs）。 |
| [智慧养老手表管理系统/oldIot/qa/issues-acceptance.cjs](../智慧养老手表管理系统/oldIot/qa/issues-acceptance.cjs) | 项目文件 智慧养老手表管理系统/oldIot/qa/issues-acceptance.cjs 的实现与配置（智慧养老手表管理系统/oldIot/qa/issues-acceptance.cjs）。 |
| [智慧养老手表管理系统/oldIot/qa/lifecycle-acceptance.cjs](../智慧养老手表管理系统/oldIot/qa/lifecycle-acceptance.cjs) | 项目文件 智慧养老手表管理系统/oldIot/qa/lifecycle-acceptance.cjs 的实现与配置（智慧养老手表管理系统/oldIot/qa/lifecycle-acceptance.cjs）。 |
| [智慧养老手表管理系统/oldIot/qa/package-lock.json](../智慧养老手表管理系统/oldIot/qa/package-lock.json) | 项目文件 智慧养老手表管理系统/oldIot/qa/package-lock.json 的实现与配置（智慧养老手表管理系统/oldIot/qa/package-lock.json）。 |
| [智慧养老手表管理系统/oldIot/qa/package.json](../智慧养老手表管理系统/oldIot/qa/package.json) | 项目文件 智慧养老手表管理系统/oldIot/qa/package.json 的实现与配置（智慧养老手表管理系统/oldIot/qa/package.json）。 |
| [智慧养老手表管理系统/oldIot/qa/run.cjs](../智慧养老手表管理系统/oldIot/qa/run.cjs) | 项目文件 智慧养老手表管理系统/oldIot/qa/run.cjs 的实现与配置（智慧养老手表管理系统/oldIot/qa/run.cjs）。 |
| [智慧养老手表管理系统/oldIot/qa/screen-acceptance.cjs](../智慧养老手表管理系统/oldIot/qa/screen-acceptance.cjs) | 项目文件 智慧养老手表管理系统/oldIot/qa/screen-acceptance.cjs 的实现与配置（智慧养老手表管理系统/oldIot/qa/screen-acceptance.cjs）。 |
| [智慧养老手表管理系统/oldIot/qa/setup-browser-acceptance.cjs](../智慧养老手表管理系统/oldIot/qa/setup-browser-acceptance.cjs) | 项目文件 智慧养老手表管理系统/oldIot/qa/setup-browser-acceptance.cjs 的实现与配置（智慧养老手表管理系统/oldIot/qa/setup-browser-acceptance.cjs）。 |
| [智慧养老手表管理系统/oldIot/src/App.vue](../智慧养老手表管理系统/oldIot/src/App.vue) | Vue 根容器与统一控件边界，集中处理焦点、错误、禁用、大字和减少动画设置。（智慧养老手表管理系统/oldIot/src/App.vue）。 |
| [智慧养老手表管理系统/oldIot/src/assets/css/accessibility.css](../智慧养老手表管理系统/oldIot/src/assets/css/accessibility.css) | 项目文件 智慧养老手表管理系统/oldIot/src/assets/css/accessibility.css 的实现与配置（智慧养老手表管理系统/oldIot/src/assets/css/accessibility.css）。 |
| [智慧养老手表管理系统/oldIot/src/assets/css/commercial.css](../智慧养老手表管理系统/oldIot/src/assets/css/commercial.css) | 工作台与集合页面布局，规范查询工具栏对齐、弹窗字段结构和底部操作间距。（智慧养老手表管理系统/oldIot/src/assets/css/commercial.css）。 |
| [智慧养老手表管理系统/oldIot/src/assets/css/element-theme.css](../智慧养老手表管理系统/oldIot/src/assets/css/element-theme.css) | 项目文件 智慧养老手表管理系统/oldIot/src/assets/css/element-theme.css 的实现与配置（智慧养老手表管理系统/oldIot/src/assets/css/element-theme.css）。 |
| [智慧养老手表管理系统/oldIot/src/assets/css/fonts.css](../智慧养老手表管理系统/oldIot/src/assets/css/fonts.css) | 声明本地英文/数字与中文衬线字体、预加载和 Unicode 分片。（智慧养老手表管理系统/oldIot/src/assets/css/fonts.css）。 |
| [智慧养老手表管理系统/oldIot/src/assets/css/icons.css](../智慧养老手表管理系统/oldIot/src/assets/css/icons.css) | 保留实际引用的兼容图标字形映射与图标字体加载规则。（智慧养老手表管理系统/oldIot/src/assets/css/icons.css）。 |
| [智慧养老手表管理系统/oldIot/src/assets/css/index.css](../智慧养老手表管理系统/oldIot/src/assets/css/index.css) | 全局页面高度、边距和基础布局规则。（智慧养老手表管理系统/oldIot/src/assets/css/index.css）。 |
| [智慧养老手表管理系统/oldIot/src/assets/css/tokens.css](../智慧养老手表管理系统/oldIot/src/assets/css/tokens.css) | 项目文件 智慧养老手表管理系统/oldIot/src/assets/css/tokens.css 的实现与配置（智慧养老手表管理系统/oldIot/src/assets/css/tokens.css）。 |
| [智慧养老手表管理系统/oldIot/src/auth.js](../智慧养老手表管理系统/oldIot/src/auth.js) | 同源会话与 CSRF 令牌管理、上传请求头和登录失效处理。（智慧养老手表管理系统/oldIot/src/auth.js）。 |
| [智慧养老手表管理系统/oldIot/src/charts.js](../智慧养老手表管理系统/oldIot/src/charts.js) | 按需图表模块与视图所属实例管理，负责响应式重绘、释放及卸载保护。（智慧养老手表管理系统/oldIot/src/charts.js）。 |
| [智慧养老手表管理系统/oldIot/src/components/DatabaseNotice.vue](../智慧养老手表管理系统/oldIot/src/components/DatabaseNotice.vue) | 数据库不可用时的就地提醒，保持当前页面可操作（智慧养老手表管理系统/oldIot/src/components/DatabaseNotice.vue）。 |
| [智慧养老手表管理系统/oldIot/src/components/FamilySlider.vue](../智慧养老手表管理系统/oldIot/src/components/FamilySlider.vue) | 家属健康、活动、位置和个人空间的响应式导航。（智慧养老手表管理系统/oldIot/src/components/FamilySlider.vue）。 |
| [智慧养老手表管理系统/oldIot/src/components/Header.vue](../智慧养老手表管理系统/oldIot/src/components/Header.vue) | 公共品牌栏、用户显示、大字开关与服务端退出。（智慧养老手表管理系统/oldIot/src/components/Header.vue）。 |
| [智慧养老手表管理系统/oldIot/src/components/NoticeEditor.vue](../智慧养老手表管理系统/oldIot/src/components/NoticeEditor.vue) | 公告标题、正文、图片上传与保存状态的共用编辑器。（智慧养老手表管理系统/oldIot/src/components/NoticeEditor.vue）。 |
| [智慧养老手表管理系统/oldIot/src/components/PositionMap.vue](../智慧养老手表管理系统/oldIot/src/components/PositionMap.vue) | 项目文件 智慧养老手表管理系统/oldIot/src/components/PositionMap.vue 的实现与配置（智慧养老手表管理系统/oldIot/src/components/PositionMap.vue）。 |
| [智慧养老手表管理系统/oldIot/src/components/RecordCards.vue](../智慧养老手表管理系统/oldIot/src/components/RecordCards.vue) | 窄屏记录卡片的字段、状态和操作槽位。（智慧养老手表管理系统/oldIot/src/components/RecordCards.vue）。 |
| [智慧养老手表管理系统/oldIot/src/components/Slider.vue](../智慧养老手表管理系统/oldIot/src/components/Slider.vue) | 管理端工作空间与档案、设备、公告菜单。（智慧养老手表管理系统/oldIot/src/components/Slider.vue）。 |
| [智慧养老手表管理系统/oldIot/src/desktop.js](../智慧养老手表管理系统/oldIot/src/desktop.js) | 前端原生初始化桥接及 F11/Escape 全屏交互（智慧养老手表管理系统/oldIot/src/desktop.js）。 |
| [智慧养老手表管理系统/oldIot/src/element.js](../智慧养老手表管理系统/oldIot/src/element.js) | 按实际模板注册 Element Plus、中文配置和 8 个字符串图标，移除旧栅格注册。（智慧养老手表管理系统/oldIot/src/element.js）。 |
| [智慧养老手表管理系统/oldIot/src/main.js](../智慧养老手表管理系统/oldIot/src/main.js) | 创建 Vue 3 应用并安装路由、页面状态、UI 组件及请求安全处理。（智慧养老手表管理系统/oldIot/src/main.js）。 |
| [智慧养老手表管理系统/oldIot/src/router.js](../智慧养老手表管理系统/oldIot/src/router.js) | 懒加载管理/家属路由，处理首次设置、登录和角色导航守卫。（智慧养老手表管理系统/oldIot/src/router.js）。 |
| [智慧养老手表管理系统/oldIot/src/store.js](../智慧养老手表管理系统/oldIot/src/store.js) | Vue reactive 的页签状态，替代旧 Vuex 的标签增删与激活。（智慧养老手表管理系统/oldIot/src/store.js）。 |
| [智慧养老手表管理系统/oldIot/src/views/Home.vue](../智慧养老手表管理系统/oldIot/src/views/Home.vue) | 管理员工作区容器，使用完整可用宽高，提供独立内容滚动、侧栏与可关闭业务标签。（智慧养老手表管理系统/oldIot/src/views/Home.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/Login.vue](../智慧养老手表管理系统/oldIot/src/views/Login.vue) | 自适应一屏的登录界面，短屏收起辅助文案，保留大字模式、表单校验与密码显隐。（智慧养老手表管理系统/oldIot/src/views/Login.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/Register.vue](../智慧养老手表管理系统/oldIot/src/views/Register.vue) | 家属凭邀请码创建关联账号。（智慧养老手表管理系统/oldIot/src/views/Register.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/Setup.vue](../智慧养老手表管理系统/oldIot/src/views/Setup.vue) | 首次启动创建本机管理员，不提供默认密码。（智慧养老手表管理系统/oldIot/src/views/Setup.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/annex.vue](../智慧养老手表管理系统/oldIot/src/views/annex.vue) | 公告展示图片和附件记录维护。（智慧养老手表管理系统/oldIot/src/views/annex.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/customer.vue](../智慧养老手表管理系统/oldIot/src/views/customer.vue) | 监护人联系资料的录入与维护。（智慧养老手表管理系统/oldIot/src/views/customer.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/device.vue](../智慧养老手表管理系统/oldIot/src/views/device.vue) | 设备查询、注册、绑定、告警处置与可见页面轮询，离开页面即停止刷新。（智慧养老手表管理系统/oldIot/src/views/device.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/family.vue](../智慧养老手表管理系统/oldIot/src/views/family.vue) | 家属账号与老人关联管理，编辑时回填家属字段并提交对应模型。（智慧养老手表管理系统/oldIot/src/views/family.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/familyHome.vue](../智慧养老手表管理系统/oldIot/src/views/familyHome.vue) | 家属页面外壳，宽屏侧栏与移动端底栏独立占位，内容滚动和路由切换复位互不遮挡。（智慧养老手表管理系统/oldIot/src/views/familyHome.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/familyMain.vue](../智慧养老手表管理系统/oldIot/src/views/familyMain.vue) | 家属照护首页；按桌面高度调整欢迎区域和公告留白，完整保留健康、活动与位置入口。（智慧养老手表管理系统/oldIot/src/views/familyMain.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/inviteCode.vue](../智慧养老手表管理系统/oldIot/src/views/inviteCode.vue) | 生成、查询和作废家属注册邀请码。（智慧养老手表管理系统/oldIot/src/views/inviteCode.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/life.vue](../智慧养老手表管理系统/oldIot/src/views/life.vue) | 生活资料查询与档案阅读，原生搜索表单适配窄屏和大字模式。（智慧养老手表管理系统/oldIot/src/views/life.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/main.vue](../智慧养老手表管理系统/oldIot/src/views/main.vue) | 运营总览、异常摘要与公告检索，公告搜索使用单层边界和原生键盘提交。（智慧养老手表管理系统/oldIot/src/views/main.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/me.vue](../智慧养老手表管理系统/oldIot/src/views/me.vue) | 当前家属资料、头像与密码维护。（智慧养老手表管理系统/oldIot/src/views/me.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/person.vue](../智慧养老手表管理系统/oldIot/src/views/person.vue) | 老人档案的新增、编辑、查询与监护关联。（智慧养老手表管理系统/oldIot/src/views/person.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/position.vue](../智慧养老手表管理系统/oldIot/src/views/position.vue) | 管理员位置查询、地图与摔倒告警，姓名输入及相邻操作按底边对齐。（智慧养老手表管理系统/oldIot/src/views/position.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/positionfamily.vue](../智慧养老手表管理系统/oldIot/src/views/positionfamily.vue) | 家属查看关联长辈最近坐标与位置状态。（智慧养老手表管理系统/oldIot/src/views/positionfamily.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/ratepre.vue](../智慧养老手表管理系统/oldIot/src/views/ratepre.vue) | 健康记录查询、异常筛选与处置，工具栏自然换行并支持键盘提交。（智慧养老手表管理系统/oldIot/src/views/ratepre.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/rateprefamily.vue](../智慧养老手表管理系统/oldIot/src/views/rateprefamily.vue) | 家属查看关联长辈的心率与血压趋势。（智慧养老手表管理系统/oldIot/src/views/rateprefamily.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/saidOperation.vue](../智慧养老手表管理系统/oldIot/src/views/saidOperation.vue) | 管理员查看、回复与删除公告评论。（智慧养老手表管理系统/oldIot/src/views/saidOperation.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/step.vue](../智慧养老手表管理系统/oldIot/src/views/step.vue) | 睡眠与步数查询、异常状态和图表详情，查询输入与操作共用单层外框。（智慧养老手表管理系统/oldIot/src/views/step.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/stepfamily.vue](../智慧养老手表管理系统/oldIot/src/views/stepfamily.vue) | 家属查看关联长辈的睡眠与步数。（智慧养老手表管理系统/oldIot/src/views/stepfamily.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/text.vue](../智慧养老手表管理系统/oldIot/src/views/text.vue) | 编辑已有公告的表单容器。（智慧养老手表管理系统/oldIot/src/views/text.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/textMain.vue](../智慧养老手表管理系统/oldIot/src/views/textMain.vue) | 家属阅读公告内容及互动。（智慧养老手表管理系统/oldIot/src/views/textMain.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/textMainUser.vue](../智慧养老手表管理系统/oldIot/src/views/textMainUser.vue) | 管理员阅读公告详情和留言。（智慧养老手表管理系统/oldIot/src/views/textMainUser.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/textOpt.vue](../智慧养老手表管理系统/oldIot/src/views/textOpt.vue) | 公告列表、编辑删除与展示图片维护，查询及发布操作统一底边和间距。（智慧养老手表管理系统/oldIot/src/views/textOpt.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/textTwo.vue](../智慧养老手表管理系统/oldIot/src/views/textTwo.vue) | 发布新公告的表单容器。（智慧养老手表管理系统/oldIot/src/views/textTwo.vue）。 |
| [智慧养老手表管理系统/oldIot/src/views/watch.vue](../智慧养老手表管理系统/oldIot/src/views/watch.vue) | 监测阈值档案的登记和维护。（智慧养老手表管理系统/oldIot/src/views/watch.vue）。 |
| [智慧养老手表管理系统/oldIot/vite.config.js](../智慧养老手表管理系统/oldIot/vite.config.js) | Vite 8 的 Vue 插件、源码别名、开发接口代理与生产构建配置。（智慧养老手表管理系统/oldIot/vite.config.js）。 |
| [智慧养老手表管理系统/watch-sim/watch-sim.config.json](../智慧养老手表管理系统/watch-sim/watch-sim.config.json) | 模拟手表的服务地址与设备凭据配置模板，使用前填入实际设备密钥。（智慧养老手表管理系统/watch-sim/watch-sim.config.json）。 |
| [智慧养老手表管理系统/watch-sim/watch-sim.js](../智慧养老手表管理系统/watch-sim/watch-sim.js) | 模拟手表登录、心跳与数据上报的 Node 联调入口；不代表真实硬件验证。（智慧养老手表管理系统/watch-sim/watch-sim.js）。 |
| [智慧养老手表管理系统/watch-sim/watch-sim.test.cjs](../智慧养老手表管理系统/watch-sim/watch-sim.test.cjs) | 项目文件 智慧养老手表管理系统/watch-sim/watch-sim.test.cjs 的实现与配置（智慧养老手表管理系统/watch-sim/watch-sim.test.cjs）。 |

## 2.0.2 重构新增模块

本段记录按职责新增的文件；详情见 [重构指引](REFACTOR_2.0.2.md)。

| 文件 | 职责 |
| --- | --- |
| [REFACTOR_2.0.2.md](../docs/REFACTOR_2.0.2.md) | 2.0.2 源码拆分边界和验收指引。 |
| [handler_alarm_process.go](../智慧养老手表管理系统/backend/internal/handlers/handler_alarm_process.go) | Go 后端按职责拆分的 alarm process 模块。 |
| [handler_device_admin.go](../智慧养老手表管理系统/backend/internal/handlers/handler_device_admin.go) | Go 后端按职责拆分的 device admin 模块。 |
| [handler_device_alert_actions.go](../智慧养老手表管理系统/backend/internal/handlers/handler_device_alert_actions.go) | Go 后端按职责拆分的 device alert actions 模块。 |
| [handler_device_alerts.go](../智慧养老手表管理系统/backend/internal/handlers/handler_device_alerts.go) | Go 后端按职责拆分的 device alerts 模块。 |
| [handler_device_inventory.go](../智慧养老手表管理系统/backend/internal/handlers/handler_device_inventory.go) | Go 后端按职责拆分的 device inventory 模块。 |
| [handler_device_parse.go](../智慧养老手表管理系统/backend/internal/handlers/handler_device_parse.go) | Go 后端按职责拆分的 device parse 模块。 |
| [handler_device_protocol.go](../智慧养老手表管理系统/backend/internal/handlers/handler_device_protocol.go) | Go 后端按职责拆分的 device protocol 模块。 |
| [handler_device_repository.go](../智慧养老手表管理系统/backend/internal/handlers/handler_device_repository.go) | Go 后端按职责拆分的 device repository 模块。 |
| [handler_device_telemetry.go](../智慧养老手表管理系统/backend/internal/handlers/handler_device_telemetry.go) | Go 后端按职责拆分的 device telemetry 模块。 |
| [handler_family_invite_join.go](../智慧养老手表管理系统/backend/internal/handlers/handler_family_invite_join.go) | Go 后端按职责拆分的 family invite join 模块。 |
| [handler_family_invites.go](../智慧养老手表管理系统/backend/internal/handlers/handler_family_invites.go) | Go 后端按职责拆分的 family invites 模块。 |
| [handler_old_person.go](../智慧养老手表管理系统/backend/internal/handlers/handler_old_person.go) | Go 后端按职责拆分的 old person 模块。 |
| [handler_position.go](../智慧养老手表管理系统/backend/internal/handlers/handler_position.go) | Go 后端按职责拆分的 position 模块。 |
| [handler_social_images.go](../智慧养老手表管理系统/backend/internal/handlers/handler_social_images.go) | Go 后端按职责拆分的 social images 模块。 |
| [handler_social_pictures.go](../智慧养老手表管理系统/backend/internal/handlers/handler_social_pictures.go) | Go 后端按职责拆分的 social pictures 模块。 |
| [handler_social_sms.go](../智慧养老手表管理系统/backend/internal/handlers/handler_social_sms.go) | Go 后端按职责拆分的 social sms 模块。 |
| [handler_social_text.go](../智慧养老手表管理系统/backend/internal/handlers/handler_social_text.go) | Go 后端按职责拆分的 social text 模块。 |
| [handler_vitals_measurements.go](../智慧养老手表管理系统/backend/internal/handlers/handler_vitals_measurements.go) | Go 后端按职责拆分的 vitals measurements 模块。 |
| [handler_vitals_ratepre.go](../智慧养老手表管理系统/backend/internal/handlers/handler_vitals_ratepre.go) | Go 后端按职责拆分的 vitals ratepre 模块。 |
| [handler_vitals_steps.go](../智慧养老手表管理系统/backend/internal/handlers/handler_vitals_steps.go) | Go 后端按职责拆分的 vitals steps 模块。 |
| [handler_watch_info.go](../智慧养老手表管理系统/backend/internal/handlers/handler_watch_info.go) | Go 后端按职责拆分的 watch info 模块。 |
| [app-components.css](../智慧养老手表管理系统/oldIot/src/styles/app-components.css) | 前端样式分层：app-components。 |
| [app-family-responsive.css](../智慧养老手表管理系统/oldIot/src/styles/app-family-responsive.css) | 前端样式分层：app-family-responsive。 |
| [app-foundation.css](../智慧养老手表管理系统/oldIot/src/styles/app-foundation.css) | 前端样式分层：app-foundation。 |
| [customer.logic.js](../智慧养老手表管理系统/oldIot/src/views/customer.logic.js) | Vue Options API 业务逻辑模块：customer.logic。 |
| [device.admin-methods.js](../智慧养老手表管理系统/oldIot/src/views/device.admin-methods.js) | Vue Options API 业务逻辑模块：device.admin-methods。 |
| [device.alert-methods.js](../智慧养老手表管理系统/oldIot/src/views/device.alert-methods.js) | Vue Options API 业务逻辑模块：device.alert-methods。 |
| [device.list-methods.js](../智慧养老手表管理系统/oldIot/src/views/device.list-methods.js) | Vue Options API 业务逻辑模块：device.list-methods。 |
| [device.logic.js](../智慧养老手表管理系统/oldIot/src/views/device.logic.js) | Vue Options API 业务逻辑模块：device.logic。 |
| [device.refresh-methods.js](../智慧养老手表管理系统/oldIot/src/views/device.refresh-methods.js) | Vue Options API 业务逻辑模块：device.refresh-methods。 |
| [device.styles.css](../智慧养老手表管理系统/oldIot/src/views/device.styles.css) | 前端样式分层：device.styles。 |
| [family.logic.js](../智慧养老手表管理系统/oldIot/src/views/family.logic.js) | Vue Options API 业务逻辑模块：family.logic。 |
| [person.logic.js](../智慧养老手表管理系统/oldIot/src/views/person.logic.js) | Vue Options API 业务逻辑模块：person.logic。 |
| [ratepre.logic.js](../智慧养老手表管理系统/oldIot/src/views/ratepre.logic.js) | Vue Options API 业务逻辑模块：ratepre.logic。 |
