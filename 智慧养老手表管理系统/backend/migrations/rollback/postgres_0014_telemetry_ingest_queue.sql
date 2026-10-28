-- Stop enqueue/drain workers before downgrading. Preserve accepted reports.
-- Old binaries ignore this table; do not delete acknowledged measurements.
DELETE FROM schema_migrations WHERE name='0014_telemetry_ingest_queue.sql';
