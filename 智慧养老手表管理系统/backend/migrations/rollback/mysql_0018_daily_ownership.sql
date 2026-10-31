-- Keep all ownership and archived snapshots when reverting application code.
DELETE FROM schema_migrations WHERE name='mysql/0018_daily_ownership.sql';
