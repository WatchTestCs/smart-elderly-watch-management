-- Never drop or truncate an unprocessed inbox as part of rollback.
DELETE FROM schema_migrations WHERE name='0014_durable_telemetry_inbox.sql';
