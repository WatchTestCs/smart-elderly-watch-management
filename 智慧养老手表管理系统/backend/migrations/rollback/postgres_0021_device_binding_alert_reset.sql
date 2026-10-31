-- Keep the corrected sentinel values when reverting application code.
DELETE FROM schema_migrations WHERE name='0021_device_binding_alert_reset.sql';
