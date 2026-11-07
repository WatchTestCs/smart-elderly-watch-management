-- Preserve explicit duty assignments when reverting application code.
-- The backfill converts an unsafe implicit privilege into durable explicit state;
-- deleting those rows would silently change authorization semantics.
DELETE FROM schema_migrations WHERE name='mysql/0022_care_duty_fail_closed.sql';
