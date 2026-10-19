# Contributing / ���빱��

This guide defines how to propose focused changes and verify them. For repository file purposes, see [docs/FILES.md](docs/FILES.md). / ���ļ�˵������ύ�۽������������֤���ֿ����ļ���;�� [�ļ�����](docs/FILES.md)��

[������Ŀ����](README.md) �� [English overview](README.en.md) �� [Support / ���ⷴ��](docs/SUPPORT.md)

Contributions can improve documentation, translations, accessibility, reproducible bug reports, or focused code fixes. Chinese and English are both welcome. Please communicate respectfully and discuss the behavior being changed rather than the person making the change.

��ӭ�Ľ��ĵ������롢�ɷ����Ժ;���ȱ�ݡ��뱣�ָĶ��۽���ʹ�ÿɸ���֤�ݣ������ز����ߡ�

## Target preview / Ŀ���֧

Base contributions on `preview` and select `preview` as the pull request base. Do not target `main` or modify release tags as part of an ordinary change. A documentation PR is not authorization to publish binaries or run release workflows.

����Ŀ��ǰά������� `preview`���ύǰ�˶� PR Ŀ�ꣻ��ͨ�ĵ����޸���Ӧ�ƶ�������ǩ���ϴ���������޸Ĳֿ����á�

## Before changing code / ��ʼǰ

1. Read the [architecture overview](architecture.md), the affected implementation, and related [ADRs](docs/decisions/README.md).
2. Check existing issues and use the [reporting templates](docs/SUPPORT.md). Discuss substantial changes to deployment, database ownership, or trust boundaries before implementing them.
3. Keep unrelated refactors, dependency upgrades, formatting, and generated files out of the change.

Preserve the product constraints: independently deployed MySQL primary and asynchronous PostgreSQL analytics; UI startup without a database; no global connection wall; no automatic replay of failed business writes; no credentials or desktop token in page code. Do not describe user-supplied Linux binaries as verified builds of the Windows source.

## Validation proportional to the change / ���Ķ�ѡ����֤

| Change | Appropriate evidence |
| --- | --- |
| Documentation or translation | Valid local links, accurate commands and version/platform descriptions; no full runtime suite needed |
| Frontend behavior | Production build and relevant browser checks, including error/empty states and narrow layouts |
| Backend behavior | Relevant Go tests and static checks; transactional behavior uses isolated real MySQL, with PostgreSQL for analytics and legacy compatibility |
| Native lifecycle or packaging | Relevant native launch, close, or installer checks; compilation alone is insufficient |

Follow [BUILD](docs/BUILD.md) and [TESTING](docs/TESTING.md). Never run automated tests against production data. Tests that skip because `TEST_MYSQL_DSN` or `TEST_DATABASE_URL` is unset are not proof of the corresponding database integration. Record what ran, what failed, what was skipped, and what remains unverified; do not repeat unrelated full suites for a small documentation change.

��֤Ӧ��Ķ�����ƥ�䡣ģ�� API ���������鲻�ܴ�����ʵ���ݿ��ԭ���������գ�δ���еĲ��Բ�����дΪͨ����

## Pull request checklist / �ύ˵��

- Describe the observable problem and resulting behavior.
- Link a reproduction or issue where relevant; explain scope and tradeoffs.
- Include concise validation evidence and relevant screenshots for UI changes.
- Update Chinese and English README sections together when changing public behavior, package names, or platform support. Detailed translated engineering docs can be added incrementally.
- Update file descriptions when adding or removing tracked files. Use the existing PR template where available, and report unavailable checks honestly.
- Keep databases, health records, tokens, connection strings, signing keys, build caches, and release binaries out of Git.

��д�����⡢�޸ĺ����Ϊ����֤�����ơ���Ҫ�ύ��ʵ�ջ����ݣ�Ҳ��ҪΪ���ü��ͨ��������豸������ɾ���������ݡ�

## Security and licensing / ��ȫ������

Use the [private security reporting guidance](.github/SECURITY.md) for exploitable vulnerabilities; do not post the exploit or sensitive data in a public issue. Do not invent a public contact address or response-time guarantee.

Respect the project [MIT license](LICENSE) and third-party notices. Include attribution required by any new asset or dependency, and explain why a new runtime dependency is needed.
