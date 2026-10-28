-- Offline PostgreSQL downgrade: retain replay proofs; old binaries ignore the table.
-- Re-applying the migration is idempotent and preserves recorded nonces.
DELETE FROM schema_migrations WHERE name='0013_device_request_nonces.sql';
