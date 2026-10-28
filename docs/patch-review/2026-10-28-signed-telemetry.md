# Signed telemetry rollout and six-file patch repair

The source patch `Careold Go Telemetry Refactor (1).txt` cannot safely replace the current server. It uses a non-Go import URL, removes existing watch routes, replaces request and response contracts, references new MySQL-only tables without PostgreSQL migrations, and acknowledges an in-memory telemetry queue before a durable commit.

## Applied, non-breaking repairs

- Preserve **existing Go handlers, /restful routes and firmware wire contracts**, including numeric strings, legacy response envelopes and the SOS binding lock order. These contracts are not dependencies on a Java runtime.
- Move malformed telemetry and SOS ID checks ahead of database transactions (earlier preview commits).
- Introduce **opt-in signed report and SOS requests**, bound to the exact JSON bytes using HMAC-SHA256, a UTC timestamp, and a fresh nonce. If a signature is supplied, it is always checked. Set `CAREOLD_REQUIRE_SIGNED_DEVICE=1` only after firmware rollout to require signatures on both endpoints.
- Persist each signed nonce in the *same transaction* as the report or a new SOS incident, keyed uniquely by device and a SHA-256 nonce digest. Durable unique keys reject concurrent replay across multiple Go instances. Failed writes roll back the nonce. Identical `requestId` SOS retries are acknowledged using the established idempotence path without duplicating incidents.
- Add **both** `migrations/0013_device_request_nonces.sql` (PostgreSQL) and `migrations/mysql/0015_device_request_nonces.sql` (MySQL), and a periodic expiry cleanup. No destructive table renames, no replacement of either driver or migration history.
- Add regression coverage for malformed signatures, expired signatures, required-signature mode, committed report replay and SOS idempotence. CI runs MySQL and PostgreSQL integration suites separately.

## HMAC contract

For the existing POST endpoints `/restful/device/report` and `/restful/device/sos`, a signed caller sends:
- `X-Device-Timestamp`: Unix seconds, with a five-minute clock-skew window.
- `X-Device-Nonce`: fresh 16–64 character ASCII token matching `[A-Za-z0-9_-]`.
- `X-Device-Signature`: lowercase or uppercase hex-encoded HMAC-SHA256 keyed by its existing device secret, over `UPPERCASE_TRIMMED_DEVICE_NO + ":" + TIMESTAMP + ":" + NONCE + ":" + hex(SHA256(raw_request_bytes))`.

Send requests over TLS. Signed telemetry repeated with the same nonce returns HTTP 409; retry with a fresh nonce. SOS retry with a *stable requestId* receives the original incident acknowledgement; a new incident press must use a new requestId. Do not enable mandatory signatures before firmware support is deployed.

## Explicitly not enabled

- **In-memory async telemetry acknowledgement**: the submitted implementation drops flush errors and can lose acknowledged medical observations. The current synchronous DB transaction is the safer complete behavior; the unsafe feature is excluded, not falsely presented as fixed or benchmarked.
- **Destructive table/route renaming**: all live clients must migrate together and both databases need backfill/rollback procedures. No such cutover is attempted here.
- **PostgreSQL as production primary**: the server startup still selects MySQL while PostgreSQL remains supported for migrations, regression and transfer. This change maintains dual-database compatibility but does not claim interchangeable primary deployment.
- **Mandatory HMAC for existing clients**: feature gating deliberately preserves currently deployed watch traffic.

## Release checks

On the isolated repair branch, require the full Go 1.26/1.27 CI matrix with real MySQL/PostgreSQL, `go test -race`, `go build`, `go vet`, `gofmt`, OpenAPI check, watch protocol, frontend, Windows sidecar and vulnerability scan before landing on `preview`.
