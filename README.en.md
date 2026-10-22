<p align="center">
  <img src="智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-192.webp" width="88" height="88" alt="Suining logo">
</p>

<h1 align="center">Suining · Elderly Care Management</h1>

<p align="center"><strong>Companionship through the years. Peace in everyday life.</strong><br>An open-source watch management platform connecting residents, families and care staff.</p>

<p align="center">
  <a href="README.md">简体中文</a> · <a href="README.en.md">English</a>
</p>

[![Preview CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml/badge.svg?branch=preview)](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml?query=branch%3Apreview) [![Preview: 2.1.0](https://img.shields.io/badge/preview-2.1.0-9b7a40)](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.1.0) [![Go: 1.23+](https://img.shields.io/badge/Go-1.23%2B-00ADD8)](智慧养老手表管理系统/backend/go.mod) [![License: MIT](https://img.shields.io/badge/license-MIT-235442)](LICENSE)

[Download](#download) · [Features](#features) · [Getting started](#getting-started) · [Architecture](#architecture) · [Source and builds](#source-and-builds) · [Validation and performance](#validation-and-performance) · [Documentation](#documentation)

Suining brings resident records, watch devices, health data, location alerts and care tasks into one workspace for care organizations and families. It uses **Tauri 2 / Rust + Vue 3 / Vite + Go**, with independently deployed **MySQL** for business data and **PostgreSQL** for asynchronous analytics.

![Suining administrator workspace: device alerts, location alerts and announcements](docs/screenshots/overview-desktop.png)

*Existing interface example with test data; this is not a native-window acceptance record for 2.1.0.*

## Download

**2.1.0 · Windows x64 · Preview channel**

| Option | Download | How to use |
| --- | --- | --- |
| Installer | [Windows Setup](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Setup-2.1.0-win-x64.exe) | Follow the installer; downloads WebView2 if missing |
| Portable | [Portable ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Portable-2.1.0-win-x64.zip) | Extract everything, run `suining.exe` beside `suining-server.exe`; requires existing WebView2 |
| Release source | [Source ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Source-2.1.0.zip) | Fixed source snapshot matching the binaries above |

Integrity and provenance: [SHA-256](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/SHA256SUMS.txt) · [Build manifest](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/release-manifest.json) · [Upgrade notes](docs/releases/2.1.0.md) · [Historical versions](docs/DOWNLOADS.md)

Database servers are not bundled. End users do not need Go, Rust or Node.js. The Windows packages are unsigned.

> [!NOTE]
> **Release packages and development source are separate.** v2.1.0 assets are fixed at [e3cf064](https://github.com/WatchTestCs/smart-elderly-watch-management/commit/e3cf06411ca99e83a4c982316327548d8d03160a); `preview` continues to change and includes later configuration diagnostics and SQL translation fixes. A branch download is not the release snapshot. Existing Release assets have not been replaced.

## Features

| Area | Capabilities |
| --- | --- |
| Residents and families | Resident, guardian and family records; invitation-based family registration |
| Watch devices | Registration, binding, authentication, online status and monitoring thresholds |
| Health and daily life | Heart rate, single-value blood pressure, sleep, steps and daily records |
| Location and alerts | Last reported location, abnormal records, SOS and event history |
| Care coordination | Accept tasks, begin care, record actions and obtain independent supervisor verification |
| Announcements | Publish illustrated announcements and manage comments in administrator and family views |
| Desktop controls | Large-text mode, keyboard access, `F11` fullscreen and narrow-screen layouts |

Typical workflow: **Create records → register and bind a watch → receive reports → create an alert task → provide care → independently verify**.

Families access only linked residents. Supervisors retain existing administration permissions; caregivers and auditors are restricted to the care endpoints with individual duty checks and their own MFA settings, with all other legacy ADMIN endpoints blocked. Historical ADMIN accounts without duty assignments remain supervisors for compatibility and must be inventoried before production rollout. Maps use Leaflet and OpenStreetMap tiles and require access to those network services.

## Getting started

### Windows desktop

1. Install Setup or fully extract the portable ZIP.
2. Prepare a separate MySQL business database; configure PostgreSQL when analytics is needed. The [dual-database Compose template](deploy/dual-db.compose.yml) exposes database ports on loopback by default.
3. Set connections and the session encryption key in the PowerShell session that launches the application:

```powershell
$env:SUINING_MYSQL_DSN='suining:REPLACE_PASSWORD@tcp(127.0.0.1:13306)/suining'
$env:SUINING_PG_ANALYTICS_URL='postgres://suining_analytics:REPLACE_PASSWORD@127.0.0.1:15432/suining_analytics?sslmode=disable'
$env:SESSION_BACKEND='database'
$env:DATA_ENCRYPTION_KEY='REPLACE_WITH_SAVED_BASE64_OF_32_RANDOM_BYTES'
& '.\suining.exe'
```

This example is for controlled local connections; remote connections require verified TLS. PG analytics can be left unconfigured, but monitor the analytics events accumulating in MySQL. Generate and back up `DATA_ENCRYPTION_KEY` at initial deployment, reuse the existing key when upgrading, and do not generate a new one on each launch. See the [backend guide](智慧养老手表管理系统/backend/README.md) for configuration details.

4. Once MySQL migrations finish, initialize the first administrator through the desktop. **There is no default phone number or universal password.** Families register with invitation codes.
5. Create resident records before registering and binding watches. Back up databases and images separately. Local application data defaults to `%APPDATA%\岁宁`.

Restart after configuration changes. Direct Go execution uses `MYSQL_DSN` / `PG_ANALYTICS_URL`; desktop connections use the `SUINING_` variables above. The old `DATABASE_URL` / `SUINING_DATABASE_URL` variables are not primary-database fallbacks.

### Local development platform

The [complete Compose platform](deploy/platform.compose.yml) includes MySQL 8.4, PostgreSQL 16, Go, Nginx and Prometheus. With Docker Compose and Python 3 installed, run from the repository root:

```sh
python3 scripts/start-platform.py
# Once services are ready, create the first administrator interactively
python3 scripts/bootstrap-admin.py
```

The default application URL is `http://127.0.0.1:8080`; metrics are at `http://127.0.0.1:9090`. The startup script creates missing files in `deploy/secrets/` and `deploy/platform.env`, preserving existing files. `--validate` also creates missing configuration, but only validates Compose without starting services. Never commit keys or environment files. See the [platform notes](docs/PLATFORM_UPGRADE_2.0.6.md).

### Upgrades and continuous care

- **Migrating a legacy PG business database:** back up databases and images, stop writes, then use `transfer-postgres` to preflight and import into an empty MySQL target. Copy images separately. Installation does not migrate data automatically; Java/H2 is outside the import tool's scope. See the [migration guide](docs/DUAL_DATABASES.md).
- **Device access and continuous care:** the desktop listens on a loopback dynamic port and stops its local Go service when closed. Continuous reporting needs a separately supervised backend, controlled network entry, notification gateway and human response procedures. See [care deployment](docs/CARE_SERVICE_DEPLOYMENT.md).

## Architecture

```mermaid
flowchart LR
  R["Tauri / Rust"] -->|"Verify, start and stop"| G["Go API and workers"]
  V["Vue 3 / WebView2"] -->|"Same-origin HTTP"| G
  V -->|"Restricted setup IPC"| R
  G -->|"Business transactions"| M[("MySQL primary")]
  M -->|"Transactional enqueue"| O["Analytics outbox"]
  O -->|"Go async projection / idempotent retry"| P[("PostgreSQL analytics")]
  G --> F["Local images and logs"]
```

MySQL is the sole business data source; PG is a rebuildable analytics projection. Vue never connects directly to databases or holds the desktop initialization token or database credentials.

| Condition | Behavior |
| --- | --- |
| MySQL unavailable or migrations incomplete | UI opens; business API returns `503 DATABASE_UNAVAILABLE` |
| PG unconfigured or unavailable | Core workflows continue; committed analytics events remain in MySQL and catch up after recovery |
| A user write fails | Preserve input and request an explicit retry; no automatic replay |
| Invalid session configuration (current `preview`) | Status reports `configError` and asks for corrected configuration and a restart |

`/healthz` reports HTTP-process liveness only. `/restful/service/status` reports database readiness and analytics connectivity. See the [overview](docs/architecture/overview.md), [implementation architecture](docs/ARCHITECTURE.md) and [dual-database guide](docs/DUAL_DATABASES.md).

## Source and builds

Use `preview` for development and the fixed source package above to reproduce a release.

```sh
git clone --branch preview https://github.com/WatchTestCs/smart-elderly-watch-management.git
cd smart-elderly-watch-management
```

```text
智慧养老手表管理系统/
├── desktop/     Tauri / Rust, native permissions and Windows packaging
├── backend/     Go API, migrations, care and analytics workers
├── oldIot/      Vue 3 / Vite, interface and browser QA
└── watch-sim/   Watch protocol simulator
deploy/         Containers, databases and operations configuration
scripts/        Platform startup, API generation and performance tools
tests/          UI smoke checks
docs/           Architecture, deployment, releases and validation
evidence/       Historical raw test evidence
```

A complete Windows build requires PowerShell 7, Node.js (`^20.19.0 || >=22.12.0`), Go 1.23+, Rust 1.85+, MSVC and Windows SDK. Module declarations and lockfiles define dependency versions.

```powershell
cd 智慧养老手表管理系统
./desktop/validate-version.ps1
./desktop/build.ps1
```

Build order: **Vue assets → embedded Go service → sidecar SHA-256 → Tauri → NSIS**. Output is under the project directory at `outputs/v2.1.0-preview`. `desktop/package-release.ps1` prepares portable/source packages and provenance. See the [project guide](智慧养老手表管理系统/README.md) for UI/API development and the [file index](docs/FILES.md) for responsibilities. `oldIot` and `careold-go` retain historical names.

## Validation and performance

| What to check | Evidence and scope |
| --- | --- |
| Current branch automation | [Preview CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml?query=branch%3Apreview): frontend tests, build and dependency audit; Go 1.23.2 / 1.27.2 tests, race detection, build and static checks with dedicated MySQL 8.4 / PG 16 |
| 2.1.0 release checks | [Release notes](docs/releases/2.1.0.md) and the attached `release-manifest.json`; use the matching commit and recorded checks |
| Backend sustained load and cache | [2026-10-22 report](docs/performance/2026-10-22-backend-hybrid/README.md), including pre-merge source, isolated environment, latency and backlog limits |
| Historical failures and raw data | [Endurance records](docs/performance/2026-10-21-go123/README.md) and [CPU / GC experiments](docs/performance/2026-10-20-go123/README.md); later results do not erase earlier failures |

Two historical 20-minute CI endurance runs failed. A later local repaired-source run completed independently; the current merged source has not rerun that same sustained workload. CPU microbenchmarks, cache hits and persisted-write measurements are different quantities. Exact numbers belong in their individual reports.

Minimum development checks, from `智慧养老手表管理系统/backend`:

```sh
go test -count=1 ./...
go vet ./...
```

For database integration, configure dedicated `TEST_MYSQL_DSN` and `TEST_DATABASE_URL`, then run `go test -race -count=1 -timeout 600s ./...`. Skipped tests without those variables are not verified. The 20-minute endurance test requires [explicit opt-in](.github/workflows/mysql-endurance.yml). See [testing requirements](docs/TESTING.md); never run automation against production databases.

Real watches, external notification gateways, production migration, remote TLS and long-term production availability still require independent acceptance. Health and location depend on valid reports; missing data does not establish normal status, and delivered notifications do not establish completed care. The project has no medical-device certification and cannot replace human emergency response.

## Troubleshooting

| Symptom | Check first |
| --- | --- |
| UI opens, but login or saving fails | MySQL connection, business database name, migration status and `SESSION_BACKEND` / `DATA_ENCRYPTION_KEY` |
| PG is offline, but writes succeed | Expected behavior; monitor analytics backlog and disk capacity |
| Records are missing after an upgrade | Offline import and separate image copying; retain original backups |
| A watch cannot reach the desktop service | The default loopback endpoint is not externally accessible; plan device access separately |
| Portable application will not start | Full extraction, both EXEs together, existing WebView2, matching asset version and SHA-256 |

## Documentation

| Purpose | References |
| --- | --- |
| Installation, versions and upgrades | [Downloads](docs/DOWNLOADS.md) · [2.1.0 notes](docs/releases/2.1.0.md) · [Changelog](CHANGELOG.md) |
| Databases, migration and backups | [Dual databases](docs/DUAL_DATABASES.md) · [Backup and restore](docs/runbooks/backup-restore.md) |
| Devices and care operations | [Care deployment](docs/CARE_SERVICE_DEPLOYMENT.md) · [Support](docs/SUPPORT.md) |
| Development and API | [Project](智慧养老手表管理系统/README.md) · [Frontend](智慧养老手表管理系统/oldIot/README.md) · [Backend](智慧养老手表管理系统/backend/README.md) · [Build](docs/BUILD.md) |
| Architecture and validation | [Overview](docs/architecture/overview.md) · [Historical decisions](docs/decisions/README.md) · [Testing](docs/TESTING.md) · [File index](docs/FILES.md) |

When reporting an [issue](https://github.com/WatchTestCs/smart-elderly-watch-management/issues), include the version or commit, OS and redacted reproduction steps. Base contributions on `preview` and follow the [contribution guide](CONTRIBUTING.md). Report vulnerabilities privately under the [security policy](.github/SECURITY.md); do not publish personal records, connection strings or keys.

[MIT License](LICENSE). Fonts, maps and other third-party components retain their respective licenses.
