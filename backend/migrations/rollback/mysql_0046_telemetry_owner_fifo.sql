-- Additive index is compatible with the previous consumer. Keep the
-- migration ledger and index to avoid rebuilding it on re-upgrade.
SELECT name FROM schema_migrations WHERE name='mysql/0046_telemetry_owner_fifo.sql';
