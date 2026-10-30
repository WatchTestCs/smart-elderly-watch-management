-- Retain observation timestamps and indexes for safe reapplication.
DELETE FROM schema_migrations WHERE name='mysql/0017_telemetry_observation_time.sql';
