-- Retain observation timestamps and indexes for safe reapplication.
DELETE FROM schema_migrations WHERE name='0015_telemetry_observation_time.sql';
