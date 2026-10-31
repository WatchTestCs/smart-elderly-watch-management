-- Preserve event metadata when reverting application code.
DELETE FROM schema_migrations WHERE name='mysql/0020_position_metadata.sql';
