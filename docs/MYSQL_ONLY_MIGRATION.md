# MySQL-only release candidate

The new backend keeps **MySQL 8.4 as the only authoritative and persistent application datastore**. It does not connect to PostgreSQL, S3, a local image directory or Redis. Browser cache, short-lived service memory, HarmonyOS encrypted offline upload queues, Prometheus metrics and operating-system files used to start the application are not alternative business databases.

## Scope and compatibility

- Preserve every existing `/api/v1/*` endpoint, the Go SOS/care workflow, device authentication, sessions, staff permissions, alert rules, telemetry, and the Vue/Tauri/HarmonyOS clients.
- Existing MySQL business tables remain authoritative: accounts (`users`, `family`), residents (`oldperson`), devices (`device`), healthcare and telemetry (`ratepre`, `watch_day_info`, `device_location`), care (`care_incident`, `care_incident_event`) and immutable `image_blob` catalog.
- **New migration 0032** introduces `image_payload`: `name VARCHAR(255)` primary key, `payload MEDIUMBLOB NOT NULL` (authenticated ciphertext), `created_at DATETIME(6)`, and age index. The catalog keeps plaintext SHA-256, size, state, creation date and reference/GC locks. Application API never returns encrypted bytes.
- **New migration 0033** retires fifteen PostgreSQL analytics-outbox triggers. The historical MySQL `analytics_outbox` and `analytics_source` tables are deliberately retained **read-only**, without a worker or new producers, to avoid destroying existing audit/export material. They can be dropped only after separate archival approval.
- Existing statistics are computed by MySQL business queries; the legacy `analytics.configured`, `analytics.ready`, `analytics.pending` response keys remain for frontend compatibility and now report same-MySQL readiness, with pending = 0. They do **not** signify a second analytics database.
- `BLOB_BACKEND=local`, `BLOB_BACKEND=s3` and nonempty `PG_ANALYTICS_URL` are rejected with an actionable configuration error. New/default `BLOB_BACKEND=mysql`. `IMAGE_DIR` and S3 credentials are no longer runtime application settings.
- `DATA_ENCRYPTION_KEY` and any rotation keyring remain necessary; do not rotate during migration.

## Required production migration (stop all old writers)

1. **Back up** the exact old MySQL schema *and data*, including `schema_migrations`, using a consistent `mysqldump --single-transaction --routines --triggers`; separately preserve the encryption keyring. Back up original local images or export the private S3 bucket into a directory with identical object names. Optionally export PostgreSQL historical projections if they must be retained, as they will not be read by this new backend.
2. **Stop the old service and workers** to prevent a write race while image objects are copied. Do not delete the original images or the PostgreSQL volume.
3. In the new checkout, set `MYSQL_DSN`, the unchanged `DATA_ENCRYPTION_KEY`, `BLOB_BACKEND=mysql`; remove `PG_ANALYTICS_URL`. Run `go run ./cmd/import-images --source /path/to/export` from the backend directory. This checks source readability and image key authentication without writing to MySQL; completeness checks run with --apply.
4. Run `go run ./cmd/import-images --source /path/to/export --apply`. The tool runs checkpointed migrations and copies plaintext-or-encrypted legacy objects into authenticated `image_payload` rows. It compares hashes and existing catalog metadata, refuses collisions and different bytes, and is restartable/idempotent. No source files are deleted.
5. A zero `missing_references` count and successful full `VerifyReferences` are required. This count covers active image references and catalog entries marked ready. Starting the API independently also checks referenced MySQL image rows before publishing business routes; missing migrated images cause a visible 503 rather than serving broken references.
6. Start `python scripts/start-platform.py` for the single-MySQL production Compose stack (or launch the Go binary directly with its MySQL DSN). Restore session/OS/runtime configuration and run the checks below.

**New installations** have no legacy objects: deploy MySQL and let the Go service apply the embedded migrations before accepting requests.

## Validation and rollback

Run the exact-source Go race suite, `go build ./...`, `go vet ./...`, the frontend tests, the 62 HarmonyOS protocol/synchronizer simulations, and native Windows/HarmonyOS acceptance separately. Integration tests must use a disposable real MySQL 8.4 with `TEST_MYSQL_DSN`.

Check on the migrated database:

```sql
SELECT COUNT(*) AS stored_images FROM image_payload;
SELECT COUNT(*) AS old_events_retained FROM analytics_outbox;
SELECT COUNT(*) AS obsolete_analytics_triggers
  FROM information_schema.TRIGGERS
  WHERE TRIGGER_SCHEMA = DATABASE() AND TRIGGER_NAME LIKE 'suining_analytics_%';
SELECT name, size_bytes, content_sha256, state FROM image_blob ORDER BY name LIMIT 20;
```

The last trigger count must be **0**, and referenced objects must pass full authenticated SHA-256 verification. Verify original and restored MySQL row counts, binding/SOS workflow and transaction rollback, backup/restore of large `MEDIUMBLOB` rows, user login, image upload/download, and database restart. An optional local image backup is a **migration input**, never a runtime fallback.

**Rollback requires restoring the complete pre-upgrade MySQL backup**, including the old migration ledger and triggers, along with the original image storage. Re-running old binaries against an upgraded schema is unsupported and will be rejected by migration checks. Do not claim a production-ready release until native and real-data recovery gates pass.
