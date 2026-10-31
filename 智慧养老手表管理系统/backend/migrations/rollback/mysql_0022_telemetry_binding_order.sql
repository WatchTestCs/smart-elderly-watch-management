-- Keep the additive queue access path when reverting application code.
DELETE FROM schema_migrations WHERE name='mysql/0022_telemetry_binding_order.sql';
