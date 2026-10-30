# Contributing / 参与贡献

[中文项目介绍](README.md) · [English overview](README.en.md) · [Support / 问题反馈](docs/SUPPORT.md)

We welcome pull requests that improve documentation, translations, accessibility, reproducible bug reports, or targeted code fixes in either Chinese or English. Please keep discussions focused on technical behavior and user impact.

欢迎提交文档改进、双语翻译、无障碍优化、可复现的缺陷报告或聚焦的代码修复。中英文交流均可，讨论请聚焦于具体功能行为与技术方案。

## Target branch / 目标分支

Base your work on `preview` and open pull requests against `preview`. Release tags and binary attachments are managed separately by maintainers during formal releases.

日常开发与维护均在 `preview` 分支进行，提交 PR 时请将目标分支选为 `preview`。

## Before changing code / 开始前准备

1. Read the [architecture overview](docs/architecture/overview.md), the relevant module README, and related [Architecture Decision Records (ADRs)](docs/decisions/README.md).
2. Check existing issues and follow the [issue templates](docs/SUPPORT.md). If you plan to change deployment topology, database responsibilities, or authentication boundaries, open an issue to discuss the design first.
3. Keep each pull request focused. Avoid bundling unrelated refactors, dependency upgrades, or formatting sweeps.

Please preserve the core architectural invariants: MySQL handles authoritative business transactions while PostgreSQL receives asynchronous analytics projections; the desktop UI starts cleanly even when databases are unreachable; failed user writes preserve form input instead of replaying automatically; and frontend code never holds database credentials or desktop initialization tokens.

## Verification requirements / 验证要求

| Change scope | Expected verification |
| --- | --- |
| Documentation or translation | Verify local markdown links, command accuracy, and version/platform descriptions |
| Frontend UI | Run a production build (`npm run build`) and relevant Playwright checks, covering empty/error states and narrow viewports |
| Go backend | Run unit tests and `go vet`; test transactional changes against an isolated MySQL instance and analytics changes against PostgreSQL |
| Desktop shell or packaging | Verify native window launch, graceful shutdown, and installer behavior in addition to compiling |

See [BUILD](docs/BUILD.md) and [TESTING](docs/TESTING.md) for environment setup. Always run automated tests against disposable test databases rather than production data.

验证范围应与改动影响面相匹配：涉及数据库事务的改动需在真实测试库上验证，涉及桌面生命周期的改动需验证原生窗口启动与退出。

## Pull request checklist / 提交清单

- State the problem clearly and describe the updated behavior.
- Link the related issue or reproduction steps, and note any design trade-offs.
- Attach test output, and include screenshots for visual or layout changes.
- Update both `README.md` and `README.en.md` whenever changing user-facing behavior, configuration variables, or build steps.
- Never commit database dumps, real resident records, tokens, connection strings, signing keys, build caches, or compiled binaries.

请在 PR 中说明问题背景、修改方案与验证结果，切勿在代码或测试截图中包含真实人员资料与敏感凭据。

## Security and licensing / 安全与开源许可

Report exploitable security vulnerabilities privately according to [.github/SECURITY.md](.github/SECURITY.md) rather than in public issues.

All contributions are licensed under the [MIT License](LICENSE). If you introduce third-party assets or libraries, ensure their licenses are compatible and properly attributed.
