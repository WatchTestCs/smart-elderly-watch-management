-- Keep all sleep observations and archived snapshots when reverting application code.
DELETE FROM schema_migrations WHERE name='mysql/0019_sleep_observation_time.sql';
