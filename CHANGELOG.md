# Changelog / 更新日志

This is a concise user-facing history, not a complete commit log. Release notes and artifact manifests retain version-specific validation details. / 本文件记录主要变化，完整验收与附件来源以相应 Release 为准。

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
