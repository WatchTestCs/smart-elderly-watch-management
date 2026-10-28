# Dual-database telemetry Repository and durable inbox

The existing synchronous device report and SOS routes retain their wire contract. Both now persist through a small Go SQL Repository that uses the existing transaction and cross-engine SQL translator.

## Opt-in nonurgent report queue

Set `CAREOLD_DURABLE_TELEMETRY=1` to enable a database-backed inbox **only when** a report supplies a fresh 1–64-character `requestId` and consists solely of steps, sleep or battery/app-version readings. Missing IDs, coordinates, heart rate, blood pressure, falls and SOS continue to use the original synchronous path.

When eligible, the report is authenticated first, the measurements are encrypted with `DATA_ENCRYPTION_KEY`, and a unique `(device_id,request_id)` row is inserted in the MySQL or PostgreSQL inbox within the same transaction as the optional signed nonce. Only after DB commit does the HTTP 200 response include `queued:true`. Duplicate request IDs are rejected with HTTP 409. Never infer `queued:true` means that day-level projections already reflect the report.

A batch worker guarded by a **database row lock** enforces FIFO across multiple Go replicas. It processes up to 32 rows per transaction, re-locks elder then device, revalidates the owner's binding epoch, and performs the existing repository write. Savepoints isolate per-item failures; failed rows remain encrypted in the database for retry or operator review. A crash rolls the entire batch transaction back. `done`, `stale` and `failed` rows are retained rather than silently dropped.

## Limitations and rollout

- Neither the old firmware nor the older dashboard is automatically switched to the queue. New clients must supply a stable requestId on retries.
- The queue is disabled by default. Verify operational queue backlog, encrypted backup/restore and database capacity before production enablement.
- Any stale-binding report is retained as `stale` and intentionally not written to the wrong elder. Operators must investigate and recover if needed.
- No asynchronous acknowledgements for SOS or abnormal vitals. Those remain synchronously durable.
- Migrations are strictly additive for MySQL and PostgreSQL. Rollback scripts retain the queue data.
- PostgreSQL primary startup remains a separate architectural limitation; these Repository queries and migrations are supported by PostgreSQL test mode, while production primary selection is currently MySQL-only.
