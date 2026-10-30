-- Keep all ownership and archived snapshots when reverting application code.
DELETE FROM schema_migrations WHERE name='0016_daily_ownership.sql';
