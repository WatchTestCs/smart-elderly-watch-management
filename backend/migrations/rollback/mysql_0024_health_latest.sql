-- Latest health snapshots are derived from health_sample and can be rebuilt.
DROP INDEX idx_health_latest_recompute ON health_sample;
DROP TABLE IF EXISTS health_latest_ready;
DROP TABLE IF EXISTS health_sample_latest;
DELETE FROM schema_migrations WHERE name='mysql/0024_health_latest.sql';
