-- Stop application workers before removing the covering query support.
DROP INDEX idx_health_rule_cover ON health_sample;
ALTER TABLE health_sample DROP COLUMN rule_source_device, DROP COLUMN rule_value, DROP COLUMN rule_legacy_unzoned;
DELETE FROM schema_migrations WHERE name='mysql/0009_rule_baseline_cover.sql';
