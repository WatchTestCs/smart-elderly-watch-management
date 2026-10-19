# Contributing / 参与贡献

[中文项目介绍](README.md) · [English overview](README.en.md) · [Support / 问题反馈](docs/SUPPORT.md)

Contributions can improve documentation, translations, accessibility, reproducible bug reports, or focused code fixes. Chinese and English are both welcome. Please communicate respectfully and discuss the behavior being changed rather than the person making the change.

欢迎改进文档、翻译、可访问性和具体缺陷。请保持改动聚焦，使用可复现证据，并尊重参与者。

## Target preview / 目标分支

Base contributions on `preview` and select `preview` as the pull request base. Do not target `main` or modify release tags as part of an ordinary change. A documentation PR is not authorization to publish binaries or run release workflows.

本项目当前维护入口是 `preview`。提交前核对 PR 目标；普通文档或修复不应移动发布标签、上传替代包或修改仓库设置。

## Before changing code / 开始前

1. Read the [architecture overview](architecture.md), the affected implementation, and related [ADRs](docs/decisions/README.md).
2. Check existing issues and use the [reporting templates](docs/SUPPORT.md). Discuss substantial changes to deployment, database ownership, or trust boundaries before implementing them.
3. Keep unrelated refactors, dependency upgrades, formatting, and generated files out of the change.

Preserve the product constraints: independently deployed MySQL primary and asynchronous PostgreSQL analytics; UI startup without a database; no global connection wall; no automatic replay of failed business writes; no credentials or desktop token in page code. Do not describe user-supplied Linux binaries as verified builds of the Windows source.

## Validation proportional to the change / 按改动选择验证

| Change | Appropriate evidence |
| --- | --- |
| Documentation or translation | Valid local links, accurate commands and version/platform descriptions; no full runtime suite needed |
| Frontend behavior | Production build and relevant browser checks, including error/empty states and narrow layouts |
| Backend behavior | Relevant Go tests and static checks; transactional behavior uses isolated real MySQL, with PostgreSQL for analytics and legacy compatibility |
| Native lifecycle or packaging | Relevant native launch, close, or installer checks; compilation alone is insufficient |

Follow [BUILD](docs/BUILD.md) and [TESTING](docs/TESTING.md). Never run automated tests against production data. Tests that skip because `TEST_MYSQL_DSN` or `TEST_DATABASE_URL` is unset are not proof of the corresponding database integration. Record what ran, what failed, what was skipped, and what remains unverified; do not repeat unrelated full suites for a small documentation change.

验证应与改动风险匹配。模拟 API 的浏览器检查不能代替真实数据库或原生窗口验收；未运行的测试不能填写为通过。

## Pull request checklist / 提交说明

- Describe the observable problem and resulting behavior.
- Link a reproduction or issue where relevant; explain scope and tradeoffs.
- Include concise validation evidence and relevant screenshots for UI changes.
- Update Chinese and English README sections together when changing public behavior, package names, or platform support. Detailed translated engineering docs can be added incrementally.
- Update file descriptions when adding or removing tracked files. Use the existing PR template where available, and report unavailable checks honestly.
- Keep databases, health records, tokens, connection strings, signing keys, build caches, and release binaries out of Git.

请写清问题、修改后的行为、验证与限制。不要提交真实照护数据，也不要为了让检查通过而清空设备归属或删除遗留数据。

## Security and licensing / 安全与许可

Use the [private security reporting guidance](.github/SECURITY.md) for exploitable vulnerabilities; do not post the exploit or sensitive data in a public issue. Do not invent a public contact address or response-time guarantee.

Respect the project [MIT license](LICENSE) and third-party notices. Include attribution required by any new asset or dependency, and explain why a new runtime dependency is needed.
