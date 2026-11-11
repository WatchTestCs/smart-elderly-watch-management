-- Stop the security alert dispatcher before rolling back the application
-- Retain audit evidence, append-only triggers and pending alerts for recovery
-- Reapplying the migration preserves these tables and their records
DELETE FROM schema_migrations WHERE name='mysql/0029_supervisor_recovery_audit.sql';
