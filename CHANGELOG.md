# Changelog / ������־

This file is the user-facing version history, not a commit log. Release notes and artifact manifests contain validation details; the complete file-purpose index is in [docs/FILES.md](docs/FILES.md). / ���ļ���¼����ʹ���ߵİ汾�仯�����ļ���;�� [�ļ�����](docs/FILES.md)�������븽����Դ����Ӧ Release��

## 2.0.6 �� UI���ջ��ջ���˫������

[�Ľ�������˵��](docs/releases/2.0.6.md)

- ͳһ��������⣬����ǰ·���۵��˵�������խ����Ƭ��������ʧ�����ԣ��Ż������������֡�
- ��������/�豸������������һ���ԡ���ֵ��������ʳҩ���ֶδ�λ��ȱ�����������
- ��ͨ SOS �ӵ������á��������ˡ������澯�رա�����ˢ�ºͷ�ҳ������
- MySQL �е�ҵ������PG ���տɿ��첽��������˫�ⲿ�𡢾ɿ⵼�뼰ͶӰ����ߡ�
- ����Զ�̽����������󶨰汾�������α���ͳ�ƣ����� MySQL�����ߵ������ PG 0006��0007��0008����֤�밲װ���߽�� [���ϼ�¼](docs/INTEGRATION_2.0.6.md)��
- ͳһӦ�á���ˡ�ǰ�˼����ļ���Ʒ�汾Ϊ 2.0.6�����ļ�¼���ؽ�����δ����Զ�� Release ���ƶ���ǩ��

## Unreleased �� documentation on preview

- Add an English README with language navigation and the same platform, database, and signing boundaries as the Chinese overview.
- Add contribution guidance, copyable issue/feature templates, support notes, and release-history navigation.
- Use static license and preview-version badges; do not imply successful CI, audit certification, or a support SLA.
- Documentation-only: no binary rebuild, new application version, or movement of existing release tags.

## 2.0.2 �� Tauri / Go and independent PostgreSQL

[Release: v2.0.2-preview.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.2-preview.1)

- Replace the Windows Electron/Java stack with Tauri 2 / Rust and a Go sidecar, retaining Vue 3 / Vite.
- Keep PostgreSQL independently deployed. Open the UI without a database; show local reminders, reconnect in the background, preserve active input, and never replay failed writes automatically.
- Keep authenticated logout available without PostgreSQL while retaining session and CSRF checks.
- Replace the originally published Windows bundle with a lightweight Setup at the user's explicit request, retaining version 2.0.2. Reuse system WebView2 and silently install it only when missing.
- Add Windows portable and exact-commit source ZIPs. Add user-supplied Linux application and database-manager attachments with explicit provenance and runtime-validation limits.
- Legacy H2 data is not converted automatically. Windows binaries are locally self-signed, without public CA trust.

## 2.0.1 �� historical Windows preview

[Release: v2.0.1-preview.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.1-preview.1) �� [Archived notes](docs/releases/2.0.1.md)

- Strengthen device-secret storage, relational integrity, deletion handling, and regression coverage.
- This release belongs to the earlier Java desktop architecture; its build and validation claims must not be reused as evidence for the current Tauri / Go implementation.

## 2.0.0 �� historical Windows preview and later Linux attachment

[Release: v2.0.0-preview.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.0-preview.1)

- Historical Windows delivery used Electron, Java, and H2, with the Vue 3 / Vite frontend.
- A user-provided Linux desktop DEB was later attached unchanged. Maintainer familiarity is limited; it may contain bugs, errors, or compatibility issues and has not undergone installation or runtime validation.

Older implementation descriptions are preserved as history and are not current setup instructions. / �ɼܹ�������ʷ������Ϊ��ǰ�������ݡ�
