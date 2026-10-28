# Durable telemetry queue — operating and safety contract

The shared `TelemetryRepository.Persist` uses the same SQL statements and transactional `db.Transaction` for MySQL and PostgreSQL. Its caller controls commit/rollback.

`CAREOLD_DURABLE_TELEMETRY=1` enables the encrypted database queue for **only** low-priority step/sleep reports without vitals, alarms, location or battery data. All other reports, SOS and fall alarms remain synchronously persisted.

- The server acknowledges `queued:true` only *after* the queue row and device heartbeat commit. The response retains the existing `returnCode` envelope and `rateState`, `preState` and `serverTime` fields.
- The row contains **no device credential** and its payload is encrypted with the existing `DATA_ENCRYPTION_KEY`.
- The worker uses `SELECT ... FOR UPDATE SKIP LOCKED` to atomically claim a record. It serializes per-device reports, locks the elder before the device, verifies the original binding, persists with the shared repository and deletes the queue row in **the same transaction**.
- Crash before commit: no ack, or the durable queue row remains. Crash during processing: both business writes and queue delete roll back; another worker retries. Failure after a savepoint rolls back partial business writes and advances `next_attempt_at`; there is no finite retry/drop threshold.
- Binding mismatch or missing owner is retained as `state='blocked'` for manual resolution. Never reassign a measurement to another elder. Review blocked rows before migration, and do not purge the queue without a separately archived, validated export.
- MySQL and PostgreSQL migrations add the same logical table with engine-specific types and indexes. No old schema or API path is removed.
- The server must have a valid encryption key. If it does not, the request transparently takes the synchronous path instead.
- Queue depth and blocked record count should be monitored in production. Do not claim a performance advantage without benchmarks.

Do not set this flag on a deployment where some old instances do not run a drain worker. Rolling back to an older binary requires draining or deliberately retaining the queued records; no destructive down migration is shipped.
