-- Keep all sleep observations and archived snapshots when reverting application code.
DELETE FROM schema_migrations WHERE name='0017_sleep_observation_time.sql';
