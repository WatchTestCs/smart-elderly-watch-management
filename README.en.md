<p align="center"><img src="智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-192.webp" width="88" height="88" alt="Suining logo"></p>

# Suining · Elder Care Management

[简体中文](README.md) | **English**

[![License: MIT](https://img.shields.io/badge/license-MIT-235442)](LICENSE) [![Preview: 2.0.2](https://img.shields.io/badge/preview-2.0.2-9b7a40)](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.2-preview.1)

**Everyday care, with clarity and calm.**

Suining (岁宁) brings resident profiles, wearable devices, health and activity records, location alerts, and service announcements into one workspace for care organizations and families. Administrators maintain care information; family members access records for the person linked to their account.

The current Windows application uses **Tauri 2 / Rust, Go, Vue 3 / Vite, and independently deployed PostgreSQL**. The interface opens without a database connection. Database-dependent operations show an explanation, while background reconnection targets the configured database. The application interface is primarily Chinese.

[Downloads](#downloads) · [Getting started](#getting-started) · [Architecture](architecture.md) · [Contributing](CONTRIBUTING.md) · [Support](docs/SUPPORT.md) · [Changelog](CHANGELOG.md)

> Active source and documentation are maintained on **preview**. Historical Java, Electron, and H2 instructions do not describe the current stack. Release source archives are pinned to their release commit; the preview branch can contain newer documentation.

## Features

| Area | Capabilities |
| --- | --- |
| Care profiles | Residents, guardians, monitoring thresholds, and device associations |
| Wearable devices | Registration, device secrets, binding codes, online status, and alert follow-up |
| Health and activity | Heart rate, single blood-pressure measurements, sleep, steps, and daily records |
| Location and safety | Most recently reported location, help requests, and alert handling |
| Family access | Invitation-based registration, linked family records, and account management |
| Communication | Announcements, image content, and comments |
| Desktop experience | Large-text controls, keyboard access, fullscreen, Setup and portable packages |

Displayed health, location, and device states depend on the most recent successful report. Missing recent data does not imply that everything is normal. Leaflet and OpenStreetMap provide map rendering; online tiles require the relevant network service.

## Downloads

Current software version: **2.0.2** · Release tag: **v2.0.2-preview.1**.

| Package | Intended use | Requirements and scope |
| --- | --- | --- |
| [Windows x64 Setup](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Setup-2.0.2-win-x64.exe) | Regular installation, about 28.8 MiB | Reuses WebView2; silently downloads and installs the shared runtime only if missing |
| [Windows x64 portable ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Portable-2.0.2-win-x64.zip) | Extract and run, about 30.7 MiB | Requires an existing WebView2 runtime; keep `suining.exe` and `suining-server.exe` together |
| [2.0.2 source ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Source-2.0.2-preview.1.zip) | Inspection and source builds | Exact release commit: `46791c2675744acd0afd4e50992522b7c3c77b81` |
| [Linux amd64 application archive](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Linux-amd64.tar.gz) | Local Go service with a browser-based interface | User-supplied binary attachment, not a Linux rebuild of the current Windows source |
| [Linux database manager DEB](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/suining-db-manager_1.0.0_amd64.deb) | PostgreSQL management tool, component version 1.0.0 | Requires GTK 3 / WebKitGTK 4.1; does not include a PostgreSQL server |

The release also contains checksums, artifact provenance, dependency notices, and a Windows source dependency inventory. Linux attachments have not undergone installation or runtime validation, and their source and migration compatibility with the current Windows implementation are not established. See [Linux package notes](docs/LINUX.md).

## Getting started

1. Choose Setup or extract the entire portable ZIP. End users do not need Go, Rust, Java, or Node.js. If WebView2 is absent, use Setup with an internet connection to install it as a shared Windows component.
2. Open the application. Without PostgreSQL, the login interface and local controls remain available; there is no blocking connection screen.
3. Have the deployment administrator prepare an independent PostgreSQL database and configure `SUINING_DATABASE_URL`. Restart the application after changing configuration. A configured database is retried automatically when connectivity recovers.
4. On a migrated empty database, create the first administrator. **There is no default phone number or universal password.** Family registration uses an administrator-issued invitation code.
5. Register residents and devices, then configure the actual device integration. The default desktop loopback service is not a public internet gateway for watches.

PostgreSQL may run as a separate local service or on an independently managed server. Auto-connect cannot infer a database address or credentials from internet access alone. Do not commit connection strings or share real credentials in issues. Detailed deployment instructions are in [SETUP](docs/SETUP.md).

### When the database is unavailable

| Situation | Behavior |
| --- | --- |
| Unavailable at startup | The interface opens; database operations explain why they cannot proceed |
| Disconnection after login | The workspace, current page, and local input remain; logout still works |
| Connection restored | Database and session state are rechecked; an expired session requires login |
| A write fails | It is not replayed automatically; the user reviews and retries it |

No fake business records or offline write queue are created. See [database availability design](docs/DATABASE_AVAILABILITY.md).

## Screenshots

![The actual Windows login interface with no database connection](docs/screenshots/tauri-2.0.2-offline.png)

![Administrator workspace using a temporary test database](docs/screenshots/tauri-2.0.2-main.png)

The interface retains the selected landscape logo, Source Serif 4, and Source Han Serif. Windows opens maximized; `F11` toggles fullscreen and `Escape` exits fullscreen. Screenshots are from the actual Tauri application and do not contain production records.

## Technology and structure

| Layer | Technology | Responsibility |
| --- | --- | --- |
| Desktop | Tauri 2, Rust, Windows WebView2 | Window, single instance, restricted native initialization, sidecar lifecycle |
| Frontend | Vue 3, Vue Router 4, Element Plus, Vite 8 | Administrator and family interfaces, forms, routing, production assets |
| Requests and visualization | Axios, ECharts 6, Leaflet | API requests, charts, and maps |
| Backend | Go standard HTTP library, pgx | Business endpoints, devices, sessions, files, and database access |
| Storage | Independent PostgreSQL, SQL transactions and embedded migrations | Business data, relational constraints, and schema changes |
| Delivery | NSIS, lockfiles, SHA-256, optional Authenticode | Windows installation, artifact integrity, and dependency records |

```text
智慧养老手表管理系统/
├── desktop/      Tauri / Rust and Windows build configuration
├── backend/      Go service, embedded migrations, and regression tests
├── oldIot/       Vue 3 / Vite UI, local fonts, and browser checks
└── watch-sim/    Device API simulator

docs/            Guides, architecture, validation records, and screenshots
```

`oldIot` and the Go module name `careold-go` are retained historical names. The current runtime does not use Electron, Java, Spring Boot, H2, or the old Webpack/node-sass toolchain.

## Build from source

Windows source builds need C++ Build Tools / Windows SDK, Rust stable, Go, Node.js, and PowerShell 7. See [BUILD](docs/BUILD.md) for versions and environment requirements. From the project directory:

```powershell
cd 智慧养老手表管理系统
./desktop/build.ps1
# Optional local self-signing; this is not publicly trusted signing
./desktop/build.ps1 -SelfSign
```

The build embeds Vue production assets and SQL migrations into the Go sidecar. Production does not require a Vite development server. Setup creation is automated; portable packaging and source archiving are separate release preparation steps.

For frontend development, use `npm ci` and `npm run dev` under `oldIot`. The API proxy targets `127.0.0.1:8795`; run the Go backend with the matching `ADDR` and a dedicated development database. Native administrator initialization is intentionally unavailable to an ordinary browser page. See the [engineering guide](智慧养老手表管理系统/README.md).

## Data and security

- Business data resides in PostgreSQL. Local images, logs, and runtime discovery files default to `%APPDATA%\岁宁`; sessions reside in Go process memory.
- Back up the database and image files separately. Legacy H2 files are **not** converted automatically to PostgreSQL.
- Native desktop initialization tokens stay outside page JavaScript. User authorization is server-side; ordinary writes use session-bound CSRF protection.
- Current Windows binaries are locally self-signed, without public CA trust or SmartScreen reputation. The application does not install a trusted root certificate.
- Recorded Windows validation covers database disconnection/recovery and desktop lifecycle. Mocked browser checks do not replace real backend tests. Linux GUI execution, all Windows versions, real wearable/SMS integration, production migrations, and formal performance benchmarks remain outside the recorded validation scope.

Report exploitable vulnerabilities privately according to the [security policy](.github/SECURITY.md). This project does not claim zero vulnerabilities or a support SLA.

## Contributing and documentation

Start with [CONTRIBUTING](CONTRIBUTING.md), target **preview**, and keep changes focused. Use [SUPPORT](docs/SUPPORT.md) for reproducible bug reports and feature requests. Both Chinese and English reports are welcome. Never include real care records, credentials, device secrets, or private keys.

| Guide | Link |
| --- | --- |
| Architecture overview and implementation | [Overview](architecture.md) · [Implementation](docs/ARCHITECTURE.md) |
| Installation and database recovery | [Setup](docs/SETUP.md) · [Availability](docs/DATABASE_AVAILABILITY.md) |
| Build and validation | [Build](docs/BUILD.md) · [Testing](docs/TESTING.md) |
| Operational notes | [Backups](docs/runbooks/backup-restore.md) · [Signing](docs/runbooks/signing.md) |
| Release history | [Changelog](CHANGELOG.md) · [2.0.2 notes](docs/releases/2.0.2.md) |
| File responsibilities and design decisions | [File index](docs/FILES.md) · [ADRs](docs/decisions/README.md) |

Most detailed engineering documents are currently in Chinese. This English README describes the same platform and database boundaries; it does not imply English localization of the application.

## License

[MIT](LICENSE). Third-party fonts, libraries, and system runtimes retain their respective licenses. Windows dependency notices and source inventories are included with the release; they do not cover the unknown build dependencies of user-supplied Linux binaries.
