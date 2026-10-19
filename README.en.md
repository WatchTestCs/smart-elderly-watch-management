<p align="center"><img src="智慧养老手表管理系统/oldIot/public/static/brand/suining-logo-192.webp" width="88" height="88" alt="Suining logo"></p>

# Suining · Elder Care Management

[简体中文](README.md) | **English**

[![License: MIT](https://img.shields.io/badge/license-MIT-235442)](LICENSE) [![Preview: 2.0.6](https://img.shields.io/badge/preview-2.0.6-9b7a40)](docs/releases/2.0.6.md)

**Everyday care, with clarity and calm.**

Suining brings resident profiles, wearable devices, health and activity records, location alerts, care tasks and announcements into one workspace for care organizations and families. Version 2.0.6 uses **Tauri 2 / Rust, Vue 3 / Vite, Go, a MySQL transactional primary and PostgreSQL analytics**. Database services are independently deployed. The application interface is primarily Chinese.

[2.0.6 release and upgrade notes](docs/releases/2.0.6.md) · [Architecture](#architecture) · [Getting started](#getting-started) · [Dual-database guide](docs/DUAL_DATABASES.md) · [Source project](智慧养老手表管理系统/README.md) · [Changelog](CHANGELOG.md)

## 2.0.6 and delivery status

Local Windows x64 Setup, portable and source packages have been built. This delivery has not been published to GitHub Releases. Their names are `Suining-Setup-2.0.6-win-x64.exe`, `Suining-Portable-2.0.6-win-x64.zip` and `Suining-Source-2.0.6.zip`; verify them against the delivered `SHA256SUMS.txt`.

- **Interface:** route-aware navigation expansion, consistent terminology, narrow-screen resident cards, loading and retry states on five management pages, and improved wide-screen layout and statistics descriptions.
- **Profiles and devices:** create residents before binding watches; profile editing preserves device ownership; retain custom thresholds, correct daily-information fields and preserve NULL for missing measurements.
- **Care workflow:** acknowledge SOS, record care actions and obtain independent verification from another supervisor; paginate tasks and history, clear the associated alert and refresh statistics.
- **Health sampling:** retain batch samples, revisions, binding isolation, incremental changes and single-source statistics with MySQL persistence. Existing UI and legacy reporting remain separate; see the [API guide](智慧养老手表管理系统/backend/docs/health-data.md).
- **Dual databases:** MySQL owns all core transactions; PostgreSQL receives asynchronous analytics projections. Offline legacy-PG import and analytics backfill tools are included.

The [historical 2.0.2 release](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.2-preview.1) remains available; its packages do not contain the 2.0.6 dual-database implementation. Historical Linux attachments have unconfirmed source and migration compatibility; see [Linux notes](docs/LINUX.md).

## Features

| Area | Capabilities |
| --- | --- |
| Residents and families | Resident, guardian and family profiles; invitation registration and family links |
| Devices | Registration, binding, unbinding, device authentication, online status and thresholds |
| Health and activity | Heart rate, single blood-pressure measurements, sleep, steps and daily records |
| Location and alerts | Latest reported location, abnormal records, SOS and event history |
| Care coordination | Acknowledgment, care actions, independent verification and service diagnostics |
| Communication | Announcements, images, comments and administration |
| Desktop interface | Large text, keyboard access, fullscreen and responsive layouts |

Supervisor, caregiver and auditor restrictions apply to care tasks; existing permissions elsewhere in administration remain unchanged. Family accounts access linked residents. Missing recent reports do not imply normal health or device status. Maps use Leaflet and OpenStreetMap tiles and require the relevant network service.

## Technology stack

Exact dependency versions are recorded in the module manifests and lockfiles.

| Layer | Technology | Responsibility |
| --- | --- | --- |
| Windows desktop | Tauri 2, Rust, WebView2 | Single instance, windows, restricted native commands and Go child-process verification/lifecycle |
| Frontend | Vue 3.5, Vue Router 4, Element Plus 2, Vite 8 | Administration and family views, forms, responsive UI and production assets |
| HTTP and visualization | Axios 1, ECharts 6, Leaflet 1.9 | API access, health charts and maps |
| Application service | Go, standard-library `net/http`, `database/sql` | Business API, devices, sessions, CSRF, images and background workers |
| Primary storage | MySQL / InnoDB, `go-sql-driver/mysql` | Business transactions, relational constraints and persistent event queues |
| Analytics storage | PostgreSQL, `pgx/v5` | Rebuildable projections, daily health and care-state summary views |
| Delivery | NSIS, npm / Cargo lockfiles, Go modules, SHA-256 | Windows packaging, dependency pinning and integrity checks |
| Verification | Go tests / race detector, Playwright, watch simulator | Business regression, browser acceptance and protocol simulation |

Handlers and background workers run within one Go service. Java, Spring Boot, Electron, H2, Vue 2 and the old Webpack build chain are not required. `oldIot` and the Go module name `careold-go` are retained historical names.

## Architecture

```mermaid
flowchart LR
  subgraph Client["Windows desktop"]
    R["Tauri / Rust"]
    V["Vue 3 / WebView2"]
    R -->|"Create window"| V
    V -->|"Restricted setup IPC"| R
  end
  subgraph Server["Go service process"]
    G["HTTP API / embedded frontend"]
    C["Care and notification worker"]
    A["Analytics worker"]
  end
  R -->|"SHA-256 check, start and stop"| G
  V -->|"Same-origin HTTP / session / CSRF"| G
  G -->|"Business transactions"| M[("MySQL / InnoDB")]
  C --> M
  M -->|"Transactional triggers"| O[("MySQL analytics outbox")]
  O -->|"Read committed events"| A
  A -->|"Async projections / idempotent retry"| P[("PostgreSQL analytics")]
  G --> F["Local images and logs"]
```

By default, Windows runs Go as a child process on a dynamic loopback port. Go serves assets and status before connecting to MySQL and applying embedded migrations. Rust validates the readiness PID and port before loading the local page. Rust manages child-process shutdown; Go also monitors its parent.

Vue calls Go over HTTP and never connects directly to either database. Initial administrator setup uses restricted Tauri IPC; Rust adds a desktop token that is not exposed to the page. Ordinary writes enforce sessions, roles and CSRF; device reporting has separate authentication. Sessions live in Go memory and do not survive service restarts.

### Database ownership and failure behavior

| Storage | Contents | Consistency and recovery |
| --- | --- | --- |
| MySQL primary | Accounts, profiles, devices, raw health reports, locations, alerts, care, audit, announcements, comments, notification and analytics queues | Sole business source of truth; all business API reads and writes use MySQL |
| PostgreSQL analytics | Health, activity, location, care-task and event projections; `daily_health` and `care_summary` views | Eventually consistent and rebuildable from MySQL; not a replacement for primary backups |
| Local files | Images, logs and runtime discovery files | Back up images separately; runtime files are not business storage |

MySQL triggers enqueue analytics events inside the originating transaction. The Go worker acknowledges an event in MySQL only after PG commits, providing at-least-once delivery. Event IDs suppress duplicate and stale updates; deletion tombstones prevent resurrection. Names, phone numbers, passwords, device secrets, care notes and full audit text are excluded from PG, but projected health and location data still require access control.

| Condition | Behavior |
| --- | --- |
| MySQL unavailable or migrations incomplete | UI opens; business API returns `503 DATABASE_UNAVAILABLE` |
| MySQL recovers | Background reconnection and session checks; expired sessions require login |
| PG unconfigured or unavailable | Core workflows continue; committed analytics events remain in MySQL, requiring queue and disk monitoring |
| PG recovers | Automatic reconnection and catch-up; no primary database failover |
| A user write fails | Preserve input and request an explicit retry; failed business requests are not replayed |

`GET /healthz` indicates HTTP-process liveness only. `GET /restful/service/status` reports primary readiness and analytics configuration/connectivity. Supervisor diagnostics additionally expose backlog and last-sync time. Notification receipts, staff arrival and completed care are separate facts.

See [implementation architecture](docs/ARCHITECTURE.md) and [dual-database operations](docs/DUAL_DATABASES.md) for process protocols, privileges, migration and recovery details.

## Getting started

1. Install the 2.0.6 Setup package, or extract the complete portable ZIP and run `suining.exe` beside `suining-server.exe`. End users do not need Go, Rust or Node.js. Setup downloads WebView2 if missing; portable use requires an existing runtime.
2. Deploy a separate MySQL business database and enable PG analytics when needed. Database servers are not bundled. The [Compose template](deploy/dual-db.compose.yml) and [environment template](deploy/dual-db.env.example) expose loopback-only database ports; start the application separately.
3. Configure the desktop environment, then launch it. Replace the placeholder passwords privately. This example is for controlled local connections; remote connections require verified TLS.

```powershell
$env:SUINING_MYSQL_DSN='suining:REPLACE_PASSWORD@tcp(127.0.0.1:13306)/suining'
$env:SUINING_PG_ANALYTICS_URL='postgres://suining_analytics:REPLACE_PASSWORD@127.0.0.1:15432/suining_analytics?sslmode=disable'
& '.\suining.exe'
```

4. Once MySQL migrations finish, create the first administrator in the empty database. **There is no default phone number or universal password.** Families register with invitation codes.
5. Create a resident, then register and bind a watch. The default desktop loopback endpoint is not an internet watch gateway. Device access requires controlled networking or a separately deployed Go service. Continuous care requires an always-on backend, notification gateway and human response procedures; see [care deployment](docs/CARE_SERVICE_DEPLOYMENT.md).

Direct Go execution uses `MYSQL_DSN` and `PG_ANALYTICS_URL`, with `ADDR=127.0.0.1:8080` by default. Desktop execution forwards the `SUINING_` variables above. Restart after configuration changes. The business service does not fall back to the old `DATABASE_URL` / `SUINING_DATABASE_URL` configuration.

### Upgrading a legacy PostgreSQL business database

Back up the database and images, stop writes from both application versions and prepare a migrated, empty MySQL target. In `backend`, set `LEGACY_POSTGRES_URL` and `MYSQL_DSN`; run `go run ./cmd/transfer-postgres` for preflight, then `go run ./cmd/transfer-postgres --apply`. The source is read-only, nonempty targets are rejected, and failed imports roll back target business rows and analytics events together. Table creation is a separate stage; copy images separately. The tool supports PG 0001–0006 and preserves the health samples, revision cursor and binding epochs from 0007/0008, not Java/H2 or automatic historical-data repair.

To rebuild lost PG analytics, prepare a new analytics database, change configuration and run `go run ./cmd/analytics-backfill --apply`. Installing the application does not migrate an old business database. Follow the [migration guide](docs/DUAL_DATABASES.md) before switching.

## Source and builds

```text
智慧养老手表管理系统/
├── desktop/      Tauri / Rust, capabilities, version checks and Windows build
├── backend/      Go API, care/analytics workers, embedded SQL and migration tools
├── oldIot/       Vue 3 / Vite, views, local fonts and browser QA
└── watch-sim/    Watch API and runtime-discovery simulator
deploy/          MySQL / PG Compose and environment templates
docs/            Configuration, architecture, verification, releases and operations
```

Windows builds require PowerShell 7, Node.js (`^20.19.0 || >=22.12.0`), Go, Rust and MSVC / Windows SDK. The Go module declares a 1.24.0 minimum; the local 2.0.6 build used Go 1.27.2. The Rust package declares a 1.85 minimum. From the source project:

```powershell
cd 智慧养老手表管理系统
./desktop/validate-version.ps1
./desktop/build.ps1
```

The pipeline builds **Vue assets → Go embedded assets, tests and binary → sidecar SHA-256 → Tauri application → NSIS Setup**. It supports `CARGO_TARGET_DIR`; Setup output is under the project’s `outputs/v2.0.6-preview`. Default builds are unsigned; optional `-SelfSign` provides local self-signing only. Portable/source ZIPs and delivery notes are separate preparation steps. See the [project README](智慧养老手表管理系统/README.md) and [backend README](智慧养老手表管理系统/backend/README.md) for development and real-database test requirements.

## Screenshots and verification

![Historical 2.0.2 login screen](docs/screenshots/tauri-2.0.2-offline.png)

![Historical 2.0.2 administrator workspace](docs/screenshots/tauri-2.0.2-main.png)

These actual Tauri screenshots are from 2.0.2 with temporary data, not 2.0.6 acceptance images. The current interface retains the landscape logo, Source Serif 4 and Source Han Serif, large-text controls, `F11` fullscreen and `Escape` to leave fullscreen.

- Dual-database development: Go tests with the race detector on isolated MySQL 8.0.46 / PostgreSQL 16.4; 11 mocked-API browser suites; 15 real core-workflow checks with PG stopped; recovery checks for zero backlog and matching analytics views.
- 2.0.6 packaging: frontend build, Go tests/vet, Tauri/NSIS build, application/installer versions, archive integrity, SHA-256 and new sidecar startup checks passed. Real dual-database integration was not rerun during packaging.
- The local 2.0.6 installer and portable application are unsigned. The installer was not executed and the native desktop window was not acceptance-tested. Historical installation results do not validate this version.
- Real watches, SMS gateways, production migration, remote TLS, long-duration load and MySQL 8.4 containers were not tested. Compose received static validation only. Mock browser and protocol checks cannot replace real backend or hardware integration.

Version-specific evidence and upgrade limits are described in the [2.0.6 notes](docs/releases/2.0.6.md). This integration preserves the remote health sampling, change-cursor and statistics interfaces on MySQL. See the [integration report](docs/INTEGRATION_2.0.6.md) for executed checks. Previously built local installers do not include this integration; rebuild from the current source.

## Documentation and contributions

| Purpose | References |
| --- | --- |
| Configuration and migration | [Dual databases](docs/DUAL_DATABASES.md) · [Setup](docs/SETUP.md) |
| Architecture and care | [Overview](architecture.md) · [Implementation](docs/ARCHITECTURE.md) · [Care deployment](docs/CARE_SERVICE_DEPLOYMENT.md) |
| Development and verification | [Project](智慧养老手表管理系统/README.md) · [Frontend](智慧养老手表管理系统/oldIot/README.md) · [Backend](智慧养老手表管理系统/backend/README.md) · [Testing](docs/TESTING.md) |
| Operations and file responsibilities | [Backup/restore](docs/runbooks/backup-restore.md) · [File index](docs/FILES.md) |
| Community and history | [Contributing](CONTRIBUTING.md) · [Support](docs/SUPPORT.md) · [Changelog](CHANGELOG.md) · [Historical decisions](docs/decisions/README.md) |

Work is restricted to `preview`; this source integration does not modify `main`, move existing tags or publish artifacts. Historical ADRs and verification documents retain their version context. Use this README, the 2.0.6 notes and the dual-database guide for current configuration.

Report reproducible problems through [Issues](https://github.com/WatchTestCs/smart-elderly-watch-management/issues); report vulnerabilities privately under the [security policy](.github/SECURITY.md). Do not disclose real personal data, database credentials or keys. [MIT License](LICENSE); fonts and third-party components retain their own licenses.
