<p align="center">
  <img src="智慧养老手表管理系统/frontend/public/static/brand/suining-logo-192.webp" width="88" height="88" alt="Suining · Elderly Care Management">
</p>

<h1 align="center">Suining · Elderly Care Management</h1>

<p align="center">Companionship through the years. Peace in everyday life.<br>An open-source watch management platform connecting residents, families, and care staff.</p>

<p align="center">
  <a href="README.md">简体中文</a> · <a href="README.en.md">English</a>
</p>

[![Preview CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml/badge.svg?branch=preview)](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml)

[Download](#download) · [Features](#features) · [Getting started](#getting-started) · [Architecture](#architecture) · [Source and builds](#source-and-builds) · [Validation and performance](#validation-and-performance) · [Documentation](#documentation)

Suining brings resident profiles, watch devices, health records, location alerts, and care tasks into a single workspace for care organizations and families. The application is built with Tauri 2, Rust, Vue 3, Vite, and Go, using an independently deployed MySQL primary database for business transactions and PostgreSQL for asynchronous analytics.

This branch also includes [SuiningWatch](SuiningWatch/README.md), a native HarmonyOS (ArkTS) watch app used to test device-to-backend protocols in the emulator. See the [watch integration notes](docs/WATCH_INTEGRATION.md) for build and test details. The watch app includes foreground heart-rate and step sensors, permission requests, device authentication, health uploads and SOS. Simulated samples remain a separate debugging source. Native SDK builds, signing and physical-watch validation use the setup described in the watch guide.

![Suining administrator workspace: device alerts, location alerts and announcements](docs/screenshots/overview-desktop.png)

*Interface preview populated with test data.*

## Download

2.1.0 · Windows x64 (Preview channel)

| Package | Download | Best for |
| --- | --- | --- |
| Installer | [Windows Setup](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Setup-2.1.0-win-x64.exe) | Standard standalone installation on a new machine |
| Portable | [Portable ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Portable-2.1.0-win-x64.zip) | Running from an extracted folder with WebView2 already installed |
| Release source | [Source ZIP](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Source-2.1.0.zip) | Reproducing the exact v2.1.0 tag including database migration scripts |

Verification: [SHA-256 checksums](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/SHA256SUMS.txt) · [Release notes](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.1.0)

The packages do not bundle a database server. End users do not need Go, Rust, or Node.js installed. The Windows binaries are currently unsigned.

> [!NOTE]
> The v2.1.0 release assets were built from commit [e3cf064](https://github.com/WatchTestCs/smart-elderly-watch-management/commit/e3cf064). To test newer changes on `preview`, build from source using the instructions below.

Use the named source archive and checksums from the Release to reproduce its build; GitHub-generated source ZIPs may have a different layout. Installers do not migrate legacy databases or local images.

## Features

| Area | What it covers |
| --- | --- |
| Residents and families | Manage resident, guardian, and family profiles; validate guardian references and guardian ages (integers from 0 to 150); protect assigned guardians from deletion; invite family members with registration codes |
| Watch devices | Register and bind watches, verify device secrets, check online status, and set alert thresholds |
| Health and daily life | View heart rate, single-value blood pressure, sleep phases, step counts, and daily care logs |
| Location and alerts | View reported locations, SOS and explicit fall types; show independent sampling context and mark unknown or historical positions |
| Care coordination | Claim alert tasks, log interventions, and require a second supervisor to verify resolution |
| Announcements | Publish illustrated notices and moderate comments across administrator and family views |
| Desktop accessibility | Toggle large-text mode, navigate by keyboard, use `F11` fullscreen, and adapt to narrow screens |

In day-to-day operation, staff create a resident profile, register and bind a watch, receive incoming telemetry, handle generated alert tasks, and have a second supervisor review completed interventions.

Access is scoped by role. Family accounts only see their linked residents. Supervisors have full administrative access, while caregivers and auditors are restricted to the care coordination workflow and their own two-factor authentication (2FA) settings.

## Getting started

### Windows desktop

1. Run the Setup installer or extract the entire portable ZIP to a local folder.
2. Start a MySQL database for business data, and optionally a PostgreSQL instance if you need analytics projections. You can use the [dual-database Compose template](deploy/dual-db.compose.yml), which exposes MySQL on port 13306 and PostgreSQL on port 15432.
3. Configure your database connection strings and session encryption key in PowerShell, then launch the app:

```powershell
$env:SUINING_MYSQL_DSN='suining:REPLACE_PASSWORD@tcp(127.0.0.1:13306)/suining'
$env:SUINING_PG_ANALYTICS_URL='postgres://suining_analytics:REPLACE_PASSWORD@127.0.0.1:15432/suining_analytics?sslmode=disable'
$env:SESSION_BACKEND='database'
$env:DATA_ENCRYPTION_KEY='REPLACE_WITH_SAVED_BASE64_OF_32_RANDOM_BYTES'
& '.\suining.exe'
```

The example above disables TLS for a local loopback setup; enable and verify TLS whenever connecting across hosts. If you omit PostgreSQL, core workflows still run normally, though you should monitor queued analytics events in MySQL. Generate `DATA_ENCRYPTION_KEY` once during initial deployment and store it safely.

4. After MySQL migrations complete, follow the desktop prompt to create the first administrator account. The system has no default phone number or master password, and family members register with invitation codes.
5. Create resident profiles before registering and binding watches. Back up your MySQL database and local image directory (`%APPDATA%\岁宁` by default) separately.

Restart the application after changing environment variables. Running the Go backend directly reads `MYSQL_DSN` and `PG_ANALYTICS_URL`, whereas launching through the desktop shell reads the `SUINING_`-prefixed variables shown above. The older `DATABASE_URL` and `SUINING_DATABASE_URL` variables are deprecated.

### Local development platform

The [platform Compose setup](deploy/platform.compose.yml) starts MySQL 8.4, PostgreSQL 16, the Go backend, Nginx, and Prometheus together:

```sh
python3 scripts/start-platform.py
# Once the containers are healthy, create the initial administrator interactively
python3 scripts/bootstrap-admin.py
```

The web UI listens at `http://127.0.0.1:8080` and Prometheus metrics are exposed at `http://127.0.0.1:9090`. On first run, the script generates `deploy/secrets/` and `deploy/platform.env`, preserving existing keys and passwords. It does not generate TLS certificates. MySQL business services start first; optional PostgreSQL startup follows and its failure does not block business services.

### Database migration and 24/7 monitoring

The desktop client binds its local Go server to a dynamic loopback port and stops the server when you close the window. For continuous watch reporting and round-the-clock SOS handling, deploy the Go backend as a persistent service behind a controlled network endpoint, connect an external notification gateway, and staff an on-call response workflow.

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

MySQL holds all authoritative business records, while PostgreSQL stores an asynchronous analytics projection that can be rebuilt from MySQL at any time. The Vue frontend communicates with the Go server over same-origin HTTP and never holds database credentials or desktop initialization tokens.

| Runtime condition | System behavior |
| --- | --- |
| MySQL is unreachable or migrations are incomplete | The UI opens normally, and business API endpoints return `503 DATABASE_UNAVAILABLE` |
| PostgreSQL is unconfigured or offline | Core workflows continue; committed analytics events stay in the MySQL outbox and sync once PG recovers |
| A user write request fails | The form keeps the user's unsaved input and prompts for a manual retry without replaying writes automatically |
| Session configuration is invalid | The status endpoint returns `configError` and prompts the operator to fix the environment and restart |

`GET /healthz` checks HTTP process liveness, while `GET /api/v1/service/status` reports database readiness and analytics sync status. See the [architecture overview](docs/ARCHITECTURE.md) for process lifecycle details and the [health data contract](智慧养老手表管理系统/backend/docs/health-data.md#跨窗口步数与夜间累计计数) for interval splitting, midnight boundaries, and alert thresholds.

## Source and builds

Active development happens on `preview`. To reproduce a published release binary, use the matching source archive from the downloads table above.

```sh
git clone --branch preview https://github.com/WatchTestCs/smart-elderly-watch-management.git
cd smart-elderly-watch-management
```

```text
智慧养老手表管理系统/
├── desktop/     Tauri / Rust shell, native permissions, and Windows packaging scripts
├── backend/     Go API, SQL migrations, care workflows, and analytics workers
├── frontend/    Vue 3 / Vite web interface and Playwright browser QA
└── watch-sim/   Watch protocol simulator
SuiningWatch/    HarmonyOS (ArkTS) wearable simulator project
deploy/          Docker Compose, database, and monitoring configurations
scripts/         Platform bootstrap, OpenAPI generator, and benchmark scripts
tests/           UI smoke tests
docs/            Architecture guides, deployment runbooks, release notes, and test logs
evidence/        Archived test outputs and fault-injection evidence
```

Building the Windows desktop package requires PowerShell 7, Node.js (`^20.19.0 || >=22.12.0`), Go (`1.26.0+` per `go.mod`), Rust 1.85+, MSVC build tools, and the Windows SDK. See [Environment setup](docs/BUILD.md) for exact version constraints.

```powershell
cd 智慧养老手表管理系统
./desktop/validate-version.ps1
./desktop/build.ps1
```

The build script compiles the Vue frontend, embeds the assets into the Go binary, generates the sidecar SHA-256 manifest, builds the Tauri shell, and packages the NSIS installer into `outputs/v2.1.0-preview`.

### Watch simulator and offline runs

The Rust watch simulator can connect to preview or generate local data without a network, backend, or account. Building it requires Rust 1.89+ and, on Windows, MSVC and the Windows SDK.

```powershell
cargo build --release --locked --manifest-path 智慧养老手表管理系统/watch-sim/Cargo.toml
.\智慧养老手表管理系统\watch-sim\target\release\watch-sim.exe --local "D:\watch-sim\local-data" --scenario demo --health-samples
```

Packaged builds include `本地模拟.cmd` for double-click startup. Each run creates a separate directory containing samples, events, and an `index.html` page that opens offline. See the [simulator guide](智慧养老手表管理系统/watch-sim/README.md) for online options, certificates, and validation commands.

Compared with the preview Node simulator, the Rust build reduced startup plus three-request time by 79.5% and peak working set by 85.2%. Coverage, adaptations and raw measurements are performance evidence and are not distributed with this repository.

## Validation

UI Quality runs with every Preview CI. `Core CI / required` covers the core checks; a formal candidate also requires native HAP compilation, actual Windows distribution tests, and complete evidence for the same SHA. See the [release gates](docs/reliability/release-governance.md) for inputs and rejection conditions.

| Area | Current reference |
| --- | --- |
| Preview branch automation | [Preview CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/workflows/ci.yml) |
| Current release notes | [Version 2.1.0](docs/releases/2.1.0.md) and the release manifest |
| Backend regression | Run `go test -count=1 ./...` and `go vet ./...` in `智慧养老手表管理系统/backend` |
| Frontend build | Run `npm ci` and `npm run build` in `智慧养老手表管理系统/frontend` |
| Watch protocol | Run `npm ci` and `npm test` in `SuiningWatch`; this does not replace native DevEco or physical-device validation |

Database integration and race tests require dedicated disposable test databases configured through `TEST_MYSQL_DSN` and `TEST_DATABASE_URL`. Tests must never target production databases. CI results must be read from the workflow run for the exact commit being assessed.

Historical performance experiments and release-stage notes are not current-version acceptance evidence. Before production deployment, separately validate physical watch hardware, external notification gateways, production data migration, network TLS, and staffed incident response. Gateway acceptance does not prove notification delivery or completed care response.
## Troubleshooting

| Symptom | What to check |
| --- | --- |
| The window opens, but login or saving fails | Verify MySQL connectivity, database name, migration status, `SESSION_BACKEND`, and `DATA_ENCRYPTION_KEY` |
| PostgreSQL is offline, but writes still succeed | This is normal; MySQL handles live transactions and queues analytics events until PG reconnects |
| A watch cannot reach the desktop app | The desktop client only listens on localhost; deploy the Go server on a reachable host for external devices |
| The portable build fails to launch | Extract the entire ZIP, keep both `.exe` files in the same directory, ensure WebView2 is installed, and verify the SHA-256 hash |

## Documentation

| Topic | Guides |
| --- | --- |
| Releases and upgrades | [2.1.0 release notes](docs/releases/2.1.0.md) · [Changelog](CHANGELOG.md) |
| Databases and backups | [Dual-database guide](docs/DUAL_DATABASES.md) · [Backup and restore runbook](docs/runbooks/backup-restore.md) |
| Operations and support | [Care service deployment](docs/CARE_SERVICE_DEPLOYMENT.md) · [Support and troubleshooting](docs/SUPPORT.md) |
| Module guides | [Workspace overview](智慧养老手表管理系统/README.md) · [Frontend](智慧养老手表管理系统/frontend/README.md) · [Backend](智慧养老手表管理系统/backend/README.md) · [Watch app](SuiningWatch/README.md) |
| Architecture and testing | [Architecture](docs/ARCHITECTURE.md) · [UI requirements](docs/UI_UX.md) · [Release gates](docs/reliability/release-governance.md) · [Testing guide](docs/TESTING.md) |

When opening an [issue](https://github.com/WatchTestCs/smart-elderly-watch-management/issues), include the application version or commit SHA, OS, network layout, and redacted logs.

Licensed under the [MIT License](LICENSE). Bundled fonts, map tiles, and third-party libraries remain under their respective licenses.

> **Breaking API change:** The `preview` branch serves API routes exclusively under `/api/v1/` and removes unversioned `/api/` aliases. Browser session cookies now use the name `SUINING_SESSION`.

## API contract

[API contracts and resolved differences](docs/API_CONTRACTS.md) list request and success fields for 109 paths. JSON success uses HTTP 200 and `returnCode="1111"`. Failures use HTTP 4xx/5xx with `returnCode="0000"`, `code`, `message` and an optional `reference`. Image downloads return bytes; uploads use multipart field `file`.

Device requests use `X-Device-Signature`, `X-Device-Timestamp` and `X-Device-Nonce`. With `SUINING_REQUIRE_SIGNED_DEVICE=1`, every device route requires a valid signature and rejects nonce replay, except `/device/report` and `/device/sos` (which reserve nonces inside their business transaction) and `/care/receipt` (which uses the gateway HMAC). Without the variable, legacy unsigned firmware still works, but any signature that is present must verify.

A `/device/report` carrying a valid `requestId` is idempotent at the transaction level: resending the same number with identical content returns the original success response (`duplicate=true`) without a second write, while reusing the number with different content returns 409. Legacy reports without a request ID keep their previous behaviour. With the durable telemetry queue enabled (`SUINING_DURABLE_TELEMETRY=1`), that path keeps its existing unique-key 409 semantics.
