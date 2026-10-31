-- Keep the corrected sentinel values when reverting application code.
DELETE FROM schema_migrations WHERE name='mysql/0023_device_binding_alert_reset.sql';
