# Careold telemetry partial-patch review (2026-10-28)

Source: user-supplied `Careold Go Telemetry Refactor (1).txt` (six-file unified diff).
Target: `preview`; retain **both MySQL and PostgreSQL** and the deployed watch protocol.

## Merge decision

| Patch area | Decision | Reason |
| --- | --- | --- |
| Typed boundary validation before DB transactions | **Adapted and merged** | The existing device JSON fields, response body, numeric-zero conventions and Go DB abstraction remain stable; invalid reports/SOS IDs now fail without DB work. |
| SOS row locking, binding-version check and idempotent retry | **Retain current implementation** | The preview code already enforces elder-before-device lock order and stable request ID retries. Replacing it would discard recent fixes. |
| RESTful-only `handler_old.go` route rewrite | **Rejected as written** | Deletes active watch/admin routes, changes verbs without rewriting handlers and includes an invalid import URL. Client/auth/CSRF/OpenAPI migration must be coordinated. |
| `telemetry_dto.go` with HMAC and memory nonce cache | **Requires redesign** | Mandatory signing breaks deployed clients; process-local, evictable nonces do not provide cross-instance replay protection, and the opt-out flag is unsafe. Design a versioned opt-in signed protocol with distributed or durable nonce state. |
| `telemetry_ingestor.go` bounded async queue | **Rejected as written** | It returns 202 before durable storage, ignores flush failures and drops single-record failures, and uses a global ingestor tied to one Handler/DB lifetime. SOS must remain synchronously durable. |
| `telemetry_repository.go` / CleanSchemaMigrationDDL | **Rejected as written** | Uses MySQL-specific `?`, `AUTO_INCREMENT`, `ON DUPLICATE KEY`, etc., while PostgreSQL compatibility is required. New tables do not match the existing migrations and cannot be adopted without dual-engine transactional migrations and data backfill. |
| Batch and HMAC tests | **Requires adaptation** | They validate unsafe semantics and a new incompatible wire format; replace with tests that assert database-free rejection and existing contracts. |

## What changed in this commit

- Extracted the existing legacy telemetry parser into a typed, validated internal structure without changing on-wire request names, response keys or SOS data writes.
- Changed both report parsing and SOS request ID validation to occur **before** the transaction starts.
- Replaced a per-request dynamic length-limits map with an immutable bounded table for a hot path.
- Added database-independent regression tests for invalid input and retained zero-metric/time/default semantics.

## Release gates

1. CI `Preview CI`: Go race tests for both supported Go versions, real MySQL + PostgreSQL, build and vet, plus OpenAPI contract checks.
2. Existing integration tests for SOS rebind/replay and watch protocol, not just happy-path mocks.
3. A separate future change must define production **PostgreSQL as a selectable primary**. PostgreSQL integration tests and migration tooling currently exist, but runtime still starts a MySQL primary.
4. No claims of throughput improvement without source-matched benchmarks; no async ACK until the queue is durably stored.

No database schema, API route, or non-Go backend modifications are introduced in this commit.
