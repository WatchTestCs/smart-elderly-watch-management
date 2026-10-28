-- Stop enqueue/drain workers before downgrading. Preserve accepted reports.
-- Do not DROP the durable queue until each row has been safely consumed.
DELETE FROM schema_migrations WHERE name='mysql/0016_telemetry_ingest_queue.sql';
