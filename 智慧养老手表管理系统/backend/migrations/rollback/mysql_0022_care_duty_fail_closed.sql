-- Keep explicit care duty assignments when reverting application code.
-- The migration only backfilled data, so rollback removes its ledger entry
-- without guessing which permission rows predated the migration.
DELETE FROM schema_migrations WHERE name='mysql/0022_care_duty_fail_closed.sql';
